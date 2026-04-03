package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Data types ───────────────────────────────────────────────────────────────

// Track is a queued audio item.
type Track struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Thumbnail string `json:"thumbnail"`
	Duration  int    `json:"duration"` // seconds; 0 = unknown
}

// MusicState is broadcast to all WebSocket clients on every change.
type MusicState struct {
	Playing   bool    `json:"playing"`
	Current   *Track  `json:"current"`
	Queue     []Track `json:"queue"`
	StartedAt *int64  `json:"startedAt"` // Unix ms when current track started; null otherwise
}

// audioClient is one connected /api/music/stream HTTP client.
type audioClient struct {
	ch chan []byte
}

// ─── Manager ──────────────────────────────────────────────────────────────────

// MusicManager handles queue, yt-dlp, ffmpeg and HTTP audio streaming.
type MusicManager struct {
	hub        *Hub
	silenceMP3 []byte // 1 s of silence, broadcast between tracks

	mu        sync.Mutex
	queue     []Track
	current   *Track
	playing   bool
	startedAt *int64     // Unix ms of current track start
	stopCh    chan struct{} // closed to skip/stop current track

	clientsMu    sync.RWMutex
	audioClients map[*audioClient]struct{}
}

func newMusicManager(hub *Hub) *MusicManager {
	m := &MusicManager{
		hub:          hub,
		audioClients: make(map[*audioClient]struct{}),
	}
	m.silenceMP3 = generateSilenceMP3()
	go m.keepAlive()
	return m
}

// generateSilenceMP3 pre-renders 1 s of silence so we can stream it between
// tracks to prevent browsers from closing the HTTP connection.
func generateSilenceMP3() []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "aevalsrc=0:sample_rate=44100:channel_layout=stereo",
		"-t", "1",
		"-f", "mp3", "-ar", "44100", "-ab", "128k",
		"pipe:1",
	)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[music] generateSilenceMP3 failed: %v", err)
		return nil
	}
	log.Printf("[music] silence buffer: %d bytes", len(out))
	return out
}

// keepAlive broadcasts silence to every connected HTTP client while no track
// is playing, so the browser never closes the audio stream connection.
func (m *MusicManager) keepAlive() {
	ticker := time.NewTicker(950 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		if m.silenceMP3 == nil {
			continue
		}
		m.mu.Lock()
		playing := m.playing
		m.mu.Unlock()
		if !playing {
			m.broadcastAudio(m.silenceMP3)
		}
	}
}

// ─── State helpers ────────────────────────────────────────────────────────────

func (m *MusicManager) snapshot() MusicState {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := make([]Track, len(m.queue))
	copy(q, m.queue)
	return MusicState{Playing: m.playing, Current: m.current, Queue: q, StartedAt: m.startedAt}
}

func (m *MusicManager) broadcast() {
	state := m.snapshot()
	payload, _ := json.Marshal(state)
	m.hub.BroadcastAll("music-state", payload)
}

// ─── Queue operations ─────────────────────────────────────────────────────────

func (m *MusicManager) Enqueue(t Track) {
	m.mu.Lock()
	m.queue = append(m.queue, t)
	shouldStart := !m.playing
	m.mu.Unlock()
	m.broadcast()
	if shouldStart {
		go m.playLoop()
	}
}

func (m *MusicManager) Remove(idx int) {
	m.mu.Lock()
	if idx >= 0 && idx < len(m.queue) {
		m.queue = append(m.queue[:idx], m.queue[idx+1:]...)
	}
	m.mu.Unlock()
	m.broadcast()
}

func (m *MusicManager) Clear() {
	m.mu.Lock()
	m.queue = nil
	m.mu.Unlock()
	m.broadcast()
}

func (m *MusicManager) Skip() {
	m.mu.Lock()
	ch := m.stopCh
	m.mu.Unlock()
	if ch != nil {
		select {
		case <-ch: // already closed
		default:
			close(ch)
		}
	}
}

// ─── Playback loop ────────────────────────────────────────────────────────────

func (m *MusicManager) playLoop() {
	for {
		m.mu.Lock()
		if len(m.queue) == 0 {
			m.playing = false
			m.current = nil
			m.stopCh = nil
			m.mu.Unlock()
			m.broadcast()
			return
		}
		track := m.queue[0]
		m.queue = m.queue[1:]
		m.current = &track
		m.playing = true
		now := time.Now().UnixMilli()
		m.startedAt = &now
		stopCh := make(chan struct{})
		m.stopCh = stopCh
		m.mu.Unlock()
		m.broadcast()

		m.streamTrack(track, stopCh)

		m.mu.Lock()
		m.startedAt = nil
		m.mu.Unlock()
	}
}

func (m *MusicManager) streamTrack(track Track, stopCh <-chan struct{}) {
	log.Printf("[music] START: %s", track.Title)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-stopCh:
			log.Printf("[music] skip requested: %s", track.Title)
			cancel()
		case <-ctx.Done():
		}
	}()

	// Use OS pipe (64 KB kernel buffer) between yt-dlp and ffmpeg.
	// io.Pipe is unbuffered and would throttle yt-dlp to ffmpeg decode speed;
	// os.Pipe lets yt-dlp burst ahead.
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		log.Printf("[music] os.Pipe: %v", err)
		return
	}

	ytArgs := ytdlpArgs(
		"-f", "bestaudio[ext=webm]/bestaudio[acodec=opus]/bestaudio[acodec=vorbis]/bestaudio",
		"-o", "-",
		"--quiet",
		track.URL,
	)
	ytCmd := exec.CommandContext(ctx, "yt-dlp", ytArgs...)
	ytCmd.Stdout = pipeW
	ytCmd.Stderr = os.Stderr // visible in docker logs

	ffCmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-vn",
		"-f", "mp3", "-ar", "44100", "-ab", "128k",
		"pipe:1",
	)
	ffCmd.Stdin = pipeR
	ffCmd.Stderr = os.Stderr

	ffOut, err := ffCmd.StdoutPipe()
	if err != nil {
		log.Printf("[music] ffmpeg StdoutPipe: %v", err)
		pipeR.Close(); pipeW.Close()
		return
	}

	if err := ffCmd.Start(); err != nil {
		log.Printf("[music] ffmpeg Start: %v", err)
		pipeR.Close(); pipeW.Close()
		return
	}
	if err := ytCmd.Start(); err != nil {
		log.Printf("[music] yt-dlp Start: %v", err)
		pipeR.Close(); pipeW.Close()
		ffCmd.Process.Kill()
		ffCmd.Wait()
		return
	}

	// When yt-dlp exits, close the write end → ffmpeg sees EOF on stdin.
	go func() {
		if err := ytCmd.Wait(); err != nil {
			log.Printf("[music] yt-dlp exit error: %v", err)
		} else {
			log.Printf("[music] yt-dlp finished OK: %s", track.Title)
		}
		pipeW.Close()
	}()

	defer func() {
		pipeR.Close()
		if err := ffCmd.Wait(); err != nil {
			log.Printf("[music] ffmpeg exit error: %v", err)
		} else {
			log.Printf("[music] ffmpeg finished OK: %s", track.Title)
		}
		log.Printf("[music] END: %s", track.Title)
	}()

	// Stream at real-time rate (128 kbps = 16 000 B/s) so playLoop stays in
	// "playing" state for the actual duration of the track.  Without throttling
	// yt-dlp dumps the whole file in seconds → the queue empties immediately
	// → playing=false → frontend destroys the <audio> element mid-playback.
	const bytesPerSec = 16_000 // 128 kbps / 8
	startTime := time.Now()
	sent := 0
	buf := make([]byte, 8192)
	for {
		n, err := ffOut.Read(buf)
		if n > 0 {
			sent += n
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			m.broadcastAudio(chunk)

			// Throttle: sleep until we are back on real-time schedule,
			// but wake early if the track is skipped (ctx cancelled).
			expected := time.Duration(float64(sent) / bytesPerSec * float64(time.Second))
			if sleep := expected - time.Since(startTime); sleep > 0 {
				select {
				case <-ctx.Done():
					log.Printf("[music] throttle interrupted (skip/stop): %s", track.Title)
					return
				case <-time.After(sleep):
				}
			}
		}
		if err != nil {
			log.Printf("[music] ffOut EOF after %d bytes: %v", sent, err)
			break
		}
	}
}

// ─── Audio broadcast ──────────────────────────────────────────────────────────

func (m *MusicManager) broadcastAudio(data []byte) {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	for c := range m.audioClients {
		select {
		case c.ch <- data:
		default: // slow client; drop frame
		}
	}
}

func (m *MusicManager) addAudioClient(c *audioClient) {
	m.clientsMu.Lock()
	m.audioClients[c] = struct{}{}
	m.clientsMu.Unlock()
}

func (m *MusicManager) removeAudioClient(c *audioClient) {
	m.clientsMu.Lock()
	delete(m.audioClients, c)
	m.clientsMu.Unlock()
}

// ─── yt-dlp helpers ───────────────────────────────────────────────────────────

// ytdlpArgs prepends common flags to every yt-dlp invocation.
func ytdlpArgs(extra ...string) []string {
	base := []string{
		"--js-runtimes", "node:/usr/bin/node",
		"--remote-components", "ejs:github", // download n-challenge solver on first run
		"--no-playlist",
	}
	return append(base, extra...)
}

// resolveTrack fetches full track info for a URL.
func resolveTrack(ctx context.Context, url string) (*Track, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	args := ytdlpArgs("--dump-json", "--quiet", url)
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	out, combinedErr := cmd.Output()
	if combinedErr != nil {
		if ee, ok := combinedErr.(*exec.ExitError); ok {
			log.Printf("[music] yt-dlp resolve stderr: %s", string(ee.Stderr))
		}
		return nil, fmt.Errorf("yt-dlp resolve: %w", combinedErr)
	}
	return parseTrackJSON(out, url), nil
}

// searchTracks runs yt-dlp ytsearch and returns up to 5 results.
func searchTracks(ctx context.Context, query string) ([]Track, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	args := ytdlpArgs("--dump-json", "--quiet", "ytsearch5:"+query)
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			log.Printf("[music] yt-dlp search stderr: %s", string(ee.Stderr))
		}
		return nil, fmt.Errorf("yt-dlp search: %w", err)
	}

	var results []Track
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var v map[string]interface{}
		if json.Unmarshal(line, &v) != nil {
			continue
		}
		id, _ := v["id"].(string)
		t := parseTrackJSON(line, "https://www.youtube.com/watch?v="+id)
		results = append(results, *t)
	}
	return results, nil
}

func parseTrackJSON(data []byte, fallbackURL string) *Track {
	var v map[string]interface{}
	if json.Unmarshal(data, &v) != nil {
		return &Track{URL: fallbackURL, Title: fallbackURL}
	}
	t := &Track{URL: fallbackURL}
	if id, ok := v["id"].(string); ok && id != "" {
		t.URL = "https://www.youtube.com/watch?v=" + id
	}
	if title, ok := v["title"].(string); ok {
		t.Title = title
	}
	if th, ok := v["thumbnail"].(string); ok {
		t.Thumbnail = th
	}
	if d, ok := v["duration"].(float64); ok {
		t.Duration = int(d)
	}
	return t
}

// ─── InnerTube search (fast, no Python overhead) ─────────────────────────────

// searchInnerTube queries YouTube's internal API directly.
// Falls back to yt-dlp via searchTracks() if this fails.
func searchInnerTube(ctx context.Context, query string) ([]Track, error) {
	ctx2, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	payload, _ := json.Marshal(map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB",
				"clientVersion": "2.20231219.01.00",
				"hl":            "en",
			},
		},
		"query":  query,
		"params": "EgIQAQ==", // videos only
	})

	req, err := http.NewRequestWithContext(ctx2, "POST",
		"https://www.youtube.com/youtubei/v1/search?prettyPrint=false",
		strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var root map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, err
	}

	// contents → twoColumnSearchResultsRenderer → primaryContents
	//          → sectionListRenderer → contents[] → itemSectionRenderer
	//          → contents[] → videoRenderer
	sections := itArr(root, "contents", "twoColumnSearchResultsRenderer",
		"primaryContents", "sectionListRenderer", "contents")

	var tracks []Track
	for _, sec := range sections {
		for _, item := range itArr(sec, "itemSectionRenderer", "contents") {
			vr := itMap(item, "videoRenderer")
			if vr == nil {
				continue
			}
			videoID := itStr(vr, "videoId")
			if videoID == "" {
				continue
			}
			title := ""
			if runs := itArr(vr, "title", "runs"); len(runs) > 0 {
				title = itStr(runs[0], "text")
			}
			thumbs := itArr(vr, "thumbnail", "thumbnails")
			thumb := ""
			if len(thumbs) > 0 {
				thumb = itStr(thumbs[len(thumbs)-1], "url")
			}
			dur := parseDurText(itStr(vr, "lengthText", "simpleText"))
			tracks = append(tracks, Track{
				URL:       "https://www.youtube.com/watch?v=" + videoID,
				Title:     title,
				Thumbnail: thumb,
				Duration:  dur,
			})
			if len(tracks) >= 5 {
				return tracks, nil
			}
		}
	}
	if len(tracks) == 0 {
		return nil, fmt.Errorf("no results")
	}
	return tracks, nil
}

// itGet navigates a nested map[string]interface{} / []interface{} by key path.
func itGet(v interface{}, keys ...string) interface{} {
	cur := v
	for _, k := range keys {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = m[k]
	}
	return cur
}
func itMap(v interface{}, keys ...string) map[string]interface{} {
	m, _ := itGet(v, keys...).(map[string]interface{})
	return m
}
func itStr(v interface{}, keys ...string) string {
	s, _ := itGet(v, keys...).(string)
	return s
}
func itArr(v interface{}, keys ...string) []interface{} {
	a, _ := itGet(v, keys...).([]interface{})
	return a
}

// parseDurText converts "3:45" or "1:03:45" → seconds.
func parseDurText(s string) int {
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	total := 0
	for _, p := range parts {
		n, _ := strconv.Atoi(strings.TrimSpace(p))
		total = total*60 + n
	}
	return total
}

// ─── HTTP handlers ────────────────────────────────────────────────────────────

// GET /api/music/stream — chunked MP3 radio stream
func (m *MusicManager) ServeStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := &audioClient{ch: make(chan []byte, 256)}
	m.addAudioClient(client)
	defer m.removeAudioClient(client)

	log.Printf("[music] stream client connected: %s", r.RemoteAddr)
	for {
		select {
		case data := <-client.ch:
			if _, err := w.Write(data); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// GET /api/music/state — current queue state as JSON
func (m *MusicManager) ServeState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.snapshot())
}

// GET /api/music/search?q=… — search YouTube (InnerTube first, yt-dlp fallback)
func (m *MusicManager) ServeSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "missing q", http.StatusBadRequest)
		return
	}

	results, err := searchInnerTube(r.Context(), q)
	if err != nil || len(results) == 0 {
		log.Printf("[music] InnerTube search failed (%v), falling back to yt-dlp", err)
		results, err = searchTracks(r.Context(), q)
		if err != nil {
			log.Printf("[music] yt-dlp search error: %v", err)
			http.Error(w, "search failed", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// POST /api/music/add  body: { url, title?, thumbnail?, duration? }
func (m *MusicManager) ServeAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body Track
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// If title not supplied, resolve via yt-dlp (blocking ~5-10 s)
	track := body
	if track.Title == "" {
		resolved, err := resolveTrack(r.Context(), body.URL)
		if err != nil {
			log.Printf("[music] resolve error: %v", err)
			http.Error(w, "could not resolve track", http.StatusBadRequest)
			return
		}
		track = *resolved
	}

	m.Enqueue(track)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(track)
}

// POST /api/music/remove  body: { index }
func (m *MusicManager) ServeRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Index int `json:"index"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	m.Remove(body.Index)
	w.WriteHeader(http.StatusOK)
}

// POST /api/music/skip
func (m *MusicManager) ServeSkip(w http.ResponseWriter, r *http.Request) {
	m.Skip()
	w.WriteHeader(http.StatusOK)
}

// POST /api/music/clear
func (m *MusicManager) ServeClear(w http.ResponseWriter, r *http.Request) {
	m.Clear()
	w.WriteHeader(http.StatusOK)
}
