package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	hub := newHub()
	go hub.run()

	music     := newMusicManager(hub)
	quiz      := newQuizManager(hub)
	musicQuiz := newMusicQuizManager(hub)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	http.HandleFunc("/ice-servers", handleIceServers)

	// Anime data proxies (Shikimori)
	http.HandleFunc("/api/quiz/animes", handleQuizAnimes)
	http.HandleFunc("/api/quiz/screenshots", handleQuizScreenshots)
	// Music themes + audio proxy (AnimeThemes.moe)
	http.HandleFunc("/api/quiz/music/themes", handleQuizMusicThemes)
	http.HandleFunc("/api/quiz/music/audio",  handleQuizMusicAudio)

	// Anime quiz game API (server-side state)
	http.HandleFunc("/api/quiz/state",        quiz.ServeState)
	http.HandleFunc("/api/quiz/lobby",         quiz.ServeOpenLobby)
	http.HandleFunc("/api/quiz/join",          quiz.ServeJoin)
	http.HandleFunc("/api/quiz/start",         quiz.ServeStart)
	http.HandleFunc("/api/quiz/answer",        quiz.ServeAnswer)
	http.HandleFunc("/api/quiz/stop",          quiz.ServeStop)
	http.HandleFunc("/api/quiz/again",         quiz.ServeAgain)

	// Music quiz game API (server-side state)
	http.HandleFunc("/api/musicquiz/animenames",  musicQuiz.ServeAnimeNames)
	http.HandleFunc("/api/musicquiz/reload-pool", musicQuiz.ServeReloadPool)
	http.HandleFunc("/api/musicquiz/state",       musicQuiz.ServeState)
	http.HandleFunc("/api/musicquiz/lobby",   musicQuiz.ServeOpenLobby)
	http.HandleFunc("/api/musicquiz/join",    musicQuiz.ServeJoin)
	http.HandleFunc("/api/musicquiz/start",   musicQuiz.ServeStart)
	http.HandleFunc("/api/musicquiz/answer",  musicQuiz.ServeAnswer)
	http.HandleFunc("/api/musicquiz/stop",    musicQuiz.ServeStop)
	http.HandleFunc("/api/musicquiz/again",   musicQuiz.ServeAgain)

	// Music bot API
	http.HandleFunc("/api/music/stream", music.ServeStream)
	http.HandleFunc("/api/music/state", music.ServeState)
	http.HandleFunc("/api/music/search", music.ServeSearch)
	http.HandleFunc("/api/music/add", music.ServeAdd)
	http.HandleFunc("/api/music/add-playlist", music.ServeAddPlaylist)
	http.HandleFunc("/api/music/remove", music.ServeRemove)
	http.HandleFunc("/api/music/skip", music.ServeSkip)
	http.HandleFunc("/api/music/clear", music.ServeClear)

	http.Handle("/", http.FileServer(http.Dir(staticDir)))

	log.Printf("ZVOnok server running on http://0.0.0.0:%s (static: %s)", port, staticDir)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleQuizAnimes proxies Shikimori anime list (avoids browser CORS restrictions).
// Supports ?page=N to load more pages.
func handleQuizAnimes(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "50"
	}

	upstream := fmt.Sprintf(
		"https://shikimori.io/api/animes?limit=%s&order=ranked&page=%s",
		limit, page,
	)

	req, err := http.NewRequest("GET", upstream, nil)
	if err != nil {
		http.Error(w, "upstream request error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "ZVOnok/1.0 (conference app)")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleQuizScreenshots proxies the Shikimori screenshots list for a single anime.
// Usage: GET /api/quiz/screenshots?id=52991
// Returns JSON array: [{ "original": "/system/animes/screenshots/...", "preview": "..." }]
func handleQuizScreenshots(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	upstream := fmt.Sprintf("https://shikimori.io/api/animes/%s/screenshots", id)

	req, err := http.NewRequest("GET", upstream, nil)
	if err != nil {
		http.Error(w, "upstream request error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "ZVOnok/1.0 (conference app)")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	// Screenshots don't change often — cache aggressively
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleQuizMusicThemes proxies the AnimeThemes.moe anime-theme list.
// Usage: GET /api/quiz/music/themes?page=N
func handleQuizMusicThemes(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}

	// Note: filter[has]=audio is NOT a valid parameter on animethemes.moe — omit it.
	upstream := fmt.Sprintf(
		"https://api.animethemes.moe/animetheme"+
			"?include=animethemeentries.videos,song,anime"+
			"&page[size]=25"+
			"&page[number]=%s",
		page,
	)

	req, err := http.NewRequest("GET", upstream, nil)
	if err != nil {
		http.Error(w, "upstream request error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ZVOnok/1.0)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleQuizMusicAudio proxies audio files from a.animethemes.moe.
// Usage: GET /api/quiz/music/audio?url=https://a.animethemes.moe/...
// Supports Range requests so the browser can seek within the audio.
func handleQuizMusicAudio(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url parameter", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		http.Error(w, "bad url", http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ZVOnok/1.0)")
	req.Header.Set("Accept", "audio/*,*/*")
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Forward relevant headers
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handleIceServers(w http.ResponseWriter, r *http.Request) {
	type iceServer struct {
		URLs       []string `json:"urls"`
		Username   string   `json:"username,omitempty"`
		Credential string   `json:"credential,omitempty"`
	}

	servers := []iceServer{
		{URLs: []string{
			"stun:stun.l.google.com:19302",
			"stun:stun1.l.google.com:19302",
		}},
	}

	turnHost := os.Getenv("TURN_HOST")
	turnUser := os.Getenv("TURN_USER")
	turnPass := os.Getenv("TURN_PASS")

	if turnHost != "" && turnUser != "" && turnPass != "" {
		servers = append(servers,
			iceServer{
				URLs:       []string{"turn:" + turnHost + ":3478?transport=udp"},
				Username:   turnUser,
				Credential: turnPass,
			},
			iceServer{
				URLs:       []string{"turn:" + turnHost + ":3478?transport=tcp"},
				Username:   turnUser,
				Credential: turnPass,
			},
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(servers)
}
