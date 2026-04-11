package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── Config ───────────────────────────────────────────────────────────────────

var musicStageDurations = []int{15, 12, 8, 8}

const (
	musicRevealPauseMs   = 5000
	musicThemePages      = 60  // /animetheme pages in parallel (25/page → up to 1500 raw themes)
	musicAnimePages      = 48  // /anime pages in parallel (100/page → ~4800 anime with MAL IDs)
	jikanTopPages        = 8   // Jikan pages, sequential (25/page → top-200 by popularity)
	jikanPageDelayMs     = 400 // ms between Jikan requests (safe for ~2.5 req/s)
)

// ─── Settings ─────────────────────────────────────────────────────────────────

type MusicQuizSettings struct {
	Rounds       int      `json:"rounds"`       // 1-20
	AllowedTypes []string `json:"allowedTypes"` // ["OP","ED","IN"] — empty = all
}

func defaultMusicSettings() MusicQuizSettings {
	return MusicQuizSettings{Rounds: 10, AllowedTypes: nil}
}

// ─── Types ────────────────────────────────────────────────────────────────────

// MusicQuizTheme is the full server-side question record.
type MusicQuizTheme struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Sequence  int    `json:"sequence"`
	SongTitle string `json:"songTitle"`
	AnimeName string `json:"animeName"`
	AudioURL  string `json:"audioUrl"`
}

// MusicRevealData is sent to clients at end of round (contains spoiler names).
type MusicRevealData struct {
	AnimeName string `json:"animeName"`
	SongTitle string `json:"songTitle"`
	Type      string `json:"type"`
	Sequence  int    `json:"sequence"`
}

// MusicQuizState is broadcast to every client on every change.
type MusicQuizState struct {
	Phase         string            `json:"phase"`
	Players       []QuizPlayer      `json:"players"`
	Scores        map[string]int    `json:"scores"`
	CurrentIdx    int               `json:"currentIdx"`
	TotalQ        int               `json:"totalQ"`
	Stage         int               `json:"stage"`
	StageTimeLeft int               `json:"stageTimeLeft"`
	RoundAnswered map[string]bool   `json:"roundAnswered"`
	RoundWinner   string            `json:"roundWinner"`
	RevealTheme   *MusicRevealData  `json:"revealTheme"`
	RoundScored   []QuizScoredEntry `json:"roundScored"`
	// Current question hints (revealed gradually)
	AudioURL    string `json:"audioUrl"`    // always shown so clients can play
	TypeBadge   string `json:"typeBadge"`   // e.g. "OP1"; "" until stage ≥ 1
	MaskedTitle string `json:"maskedTitle"` // letter-masked animeName; "" until stage ≥ 2
	// Settings & pool status (always present, even when idle)
	Settings     MusicQuizSettings `json:"settings"`
	PoolReady    bool              `json:"poolReady"`
	PoolCount    int               `json:"poolCount"`
	PoolPopular  int               `json:"poolPopular"`  // themes from popular anime (MAL-matched)
	PoolError    string            `json:"poolError,omitempty"`
}

// ─── Manager ──────────────────────────────────────────────────────────────────

type MusicQuizManager struct {
	hub *Hub

	// Game state (protected by mu)
	mu            sync.Mutex
	phase         string
	players       []QuizPlayer
	scores        map[string]int
	questions     []MusicQuizTheme // selected subset for current game
	settings      MusicQuizSettings
	currentIdx    int
	stage         int
	stageTimeLeft int
	roundAnswered map[string]bool
	roundWinner   string
	revealTheme   *MusicRevealData
	roundScored   []QuizScoredEntry
	stopCh        chan struct{}

	// Theme pool (protected by poolMu, populated at startup)
	poolMu         sync.RWMutex
	themePool      []MusicQuizTheme // sorted: popular (MAL-matched) first, then unmatched
	poolPopular    int              // how many themes at the start of themePool have known MAL rank
	poolReady      bool
	poolError      string
}

func newMusicQuizManager(hub *Hub) *MusicQuizManager {
	m := &MusicQuizManager{
		hub:           hub,
		phase:         "idle",
		settings:      defaultMusicSettings(),
		scores:        map[string]int{},
		roundAnswered: map[string]bool{},
		roundScored:   []QuizScoredEntry{},
	}
	go m.fetchPool()
	return m
}

// ─── Pool management ──────────────────────────────────────────────────────────

// rawAnimethemeEntry matches the animethemes.moe /animetheme JSON for one theme.
type rawAnimethemeEntry struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Sequence *int   `json:"sequence"`
	Anime    struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"anime"`
	Song struct {
		Title string `json:"title"`
	} `json:"song"`
	AnimethemeEntries []struct {
		Spoiler bool `json:"spoiler"`
		Videos  []struct {
			Link string `json:"link"`
		} `json:"videos"`
	} `json:"animethemeentries"`
}

// rawAnimeEntry matches the animethemes.moe /anime JSON (with resources included).
type rawAnimeEntry struct {
	ID        int `json:"id"`
	Resources []struct {
		ExternalID int    `json:"external_id"`
		Site       string `json:"site"`
	} `json:"resources"`
}

func videoToAudioProxyURL(videoLink string) string {
	if videoLink == "" {
		return ""
	}
	audioURL := strings.Replace(videoLink, "//v.animethemes.moe/", "//a.animethemes.moe/", 1)
	if strings.HasSuffix(audioURL, ".webm") {
		audioURL = audioURL[:len(audioURL)-5] + ".ogg"
	}
	return "/api/quiz/music/audio?url=" + url.QueryEscape(audioURL)
}

// fetchThemePage fetches one page from /animetheme (no resources — just anime.id).
func fetchThemePage(client *http.Client, page int) []rawAnimethemeEntry {
	apiURL := fmt.Sprintf(
		"https://api.animethemes.moe/animetheme"+
			"?include=animethemeentries.videos,song,anime"+
			"&page[size]=25"+
			"&page[number]=%d",
		page,
	)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ZVOnok/1.0)")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	var body struct {
		Entries []rawAnimethemeEntry `json:"animethemes"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	return body.Entries
}

// fetchAnimePage fetches one page from /anime?include=resources (100 anime per page).
// Returns a map slice of animethemesAnimeID → MAL_ID.
func fetchAnimePage(client *http.Client, page int) []rawAnimeEntry {
	apiURL := fmt.Sprintf(
		"https://api.animethemes.moe/anime"+
			"?include=resources"+
			"&page[size]=100"+
			"&page[number]=%d",
		page,
	)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ZVOnok/1.0)")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	var body struct {
		Anime []rawAnimeEntry `json:"anime"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	return body.Anime
}

// fetchJikanTopRanks returns map[malID]popularityRank (rank 1 = most popular).
func fetchJikanTopRanks(client *http.Client, pages int) map[int]int {
	ranks := make(map[int]int)
	for page := 1; page <= pages; page++ {
		if page > 1 {
			time.Sleep(time.Duration(jikanPageDelayMs) * time.Millisecond)
		}
		apiURL := fmt.Sprintf(
			"https://api.jikan.moe/v4/top/anime?type=tv&filter=bypopularity&limit=25&page=%d",
			page,
		)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ZVOnok/1.0)")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		var body struct {
			Data []struct {
				MalID int `json:"mal_id"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		for i, a := range body.Data {
			if a.MalID > 0 {
				ranks[a.MalID] = (page-1)*25 + i + 1
			}
		}
	}
	return ranks
}

// poolTheme is used only during pool construction.
type poolTheme struct {
	theme MusicQuizTheme
	rank  int // MAL popularity rank; 0 = no match (goes to end of pool)
}

func (m *MusicQuizManager) fetchPool() {
	client := &http.Client{Timeout: 20 * time.Second}

	// ── A: Fetch Jikan top ranks (sequential, rate-limited) ───────────────────
	ranksCh := make(chan map[int]int, 1)
	go func() { ranksCh <- fetchJikanTopRanks(client, jikanTopPages) }()

	// ── B: Fetch animethemes /anime pages to build animeID→malID map ──────────
	type animeIDResult struct{ entries []rawAnimeEntry }
	animeCh := make(chan []rawAnimeEntry, musicAnimePages)
	for page := 1; page <= musicAnimePages; page++ {
		go func(p int) { animeCh <- fetchAnimePage(client, p) }(page)
	}
	animeIDtoMAL := make(map[int]int) // animethemes anime.id → mal_id
	for i := 0; i < musicAnimePages; i++ {
		for _, a := range <-animeCh {
			for _, r := range a.Resources {
				if r.Site == "MyAnimeList" && r.ExternalID > 0 {
					animeIDtoMAL[a.ID] = r.ExternalID
					break
				}
			}
		}
	}

	// ── C: Fetch animetheme theme pages in parallel ────────────────────────────
	themeCh := make(chan []rawAnimethemeEntry, musicThemePages)
	for page := 1; page <= musicThemePages; page++ {
		go func(p int) { themeCh <- fetchThemePage(client, p) }(page)
	}
	var rawEntries []rawAnimethemeEntry
	for i := 0; i < musicThemePages; i++ {
		rawEntries = append(rawEntries, <-themeCh...)
	}

	// ── D: Wait for Jikan, then join ranks ────────────────────────────────────
	malRanks := <-ranksCh // map[malID]rank

	// ── E: Build pool items with rank ─────────────────────────────────────────
	seen := make(map[int]bool)
	var poolItems []poolTheme
	for _, e := range rawEntries {
		if seen[e.ID] || len(e.AnimethemeEntries) == 0 {
			continue
		}
		entry := e.AnimethemeEntries[0]
		if entry.Spoiler || len(entry.Videos) == 0 {
			continue
		}
		audioProxy := videoToAudioProxyURL(entry.Videos[0].Link)
		if audioProxy == "" {
			continue
		}
		seen[e.ID] = true

		seq := 1
		if e.Sequence != nil && *e.Sequence > 0 {
			seq = *e.Sequence
		}
		themeType := strings.ToUpper(e.Type)
		if themeType == "" {
			themeType = "OP"
		}

		rank := 0
		if malID := animeIDtoMAL[e.Anime.ID]; malID > 0 {
			rank = malRanks[malID] // 0 if not in top list
		}

		poolItems = append(poolItems, poolTheme{
			rank: rank,
			theme: MusicQuizTheme{
				ID:        e.ID,
				Type:      themeType,
				Sequence:  seq,
				SongTitle: e.Song.Title,
				AnimeName: e.Anime.Name,
				AudioURL:  audioProxy,
			},
		})
	}

	// ── F: Sort popular first ─────────────────────────────────────────────────
	sort.SliceStable(poolItems, func(i, j int) bool {
		ri, rj := poolItems[i].rank, poolItems[j].rank
		if ri == 0 {
			return false
		}
		if rj == 0 {
			return true
		}
		return ri < rj
	})

	all := make([]MusicQuizTheme, len(poolItems))
	popularCount := 0
	for i, p := range poolItems {
		all[i] = p.theme
		if p.rank > 0 {
			popularCount++
		}
	}

	m.poolMu.Lock()
	if len(all) > 0 {
		m.themePool = all
		m.poolPopular = popularCount
		m.poolReady = true
	} else {
		m.poolError = "Не удалось загрузить треки от AnimeThemes"
	}
	m.poolMu.Unlock()
	m.broadcast()
}

// poolSnapshot returns current pool status (safe to call without mu).
func (m *MusicQuizManager) poolSnapshot() (ready bool, count, popular int, errStr string) {
	m.poolMu.RLock()
	defer m.poolMu.RUnlock()
	return m.poolReady, len(m.themePool), m.poolPopular, m.poolError
}

// ─── State helpers ────────────────────────────────────────────────────────────

func (m *MusicQuizManager) snapshot() MusicQuizState {
	m.mu.Lock()
	defer m.mu.Unlock()

	ps := make([]QuizPlayer, len(m.players))
	copy(ps, m.players)
	sc := make(map[string]int, len(m.scores))
	for k, v := range m.scores {
		sc[k] = v
	}
	ra := make(map[string]bool, len(m.roundAnswered))
	for k, v := range m.roundAnswered {
		ra[k] = v
	}
	scored := make([]QuizScoredEntry, len(m.roundScored))
	copy(scored, m.roundScored)

	poolReady, poolCount, poolPopular, poolErr := m.poolSnapshot()

	state := MusicQuizState{
		Phase:         m.phase,
		Players:       ps,
		Scores:        sc,
		CurrentIdx:    m.currentIdx,
		Stage:         m.stage,
		StageTimeLeft: m.stageTimeLeft,
		RoundAnswered: ra,
		RoundWinner:   m.roundWinner,
		RevealTheme:   m.revealTheme,
		RoundScored:   scored,
		Settings:      m.settings,
		PoolReady:     poolReady,
		PoolCount:     poolCount,
		PoolPopular:   poolPopular,
		PoolError:     poolErr,
	}
	if len(m.questions) > 0 {
		state.TotalQ = len(m.questions)
		if m.currentIdx < len(m.questions) {
			cur := m.questions[m.currentIdx]
			state.AudioURL = cur.AudioURL
			// TypeBadge visible from stage 1
			if m.stage >= 1 {
				typeName := cur.Type
				if typeName == "" {
					typeName = "OP"
				}
				seq := cur.Sequence
				if seq == 0 {
					seq = 1
				}
				state.TypeBadge = typeName + string(rune('0'+seq))
				if seq > 9 {
					state.TypeBadge = typeName
				}
			}
			// MaskedTitle visible from stage 2
			if m.stage >= 2 {
				state.MaskedTitle = maskedTitle(cur.AnimeName, cur.ID, m.stage-1)
			}
		}
	}
	return state
}

func (m *MusicQuizManager) broadcast() {
	state := m.snapshot()
	payload, _ := json.Marshal(state)
	m.hub.BroadcastAll("musicquiz-state", payload)
}

// ─── Timer ────────────────────────────────────────────────────────────────────

func (m *MusicQuizManager) stopTimer() {
	m.mu.Lock()
	if m.stopCh != nil {
		select {
		case <-m.stopCh:
		default:
			close(m.stopCh)
		}
		m.stopCh = nil
	}
	m.mu.Unlock()
}

func (m *MusicQuizManager) startStageTimer(s int) {
	m.stopTimer()
	m.mu.Lock()
	m.stage = s
	m.stageTimeLeft = musicStageDurations[s]
	stopCh := make(chan struct{})
	m.stopCh = stopCh
	m.mu.Unlock()
	m.broadcast()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				m.mu.Lock()
				m.stageTimeLeft--
				remaining := m.stageTimeLeft
				curStage := m.stage
				m.mu.Unlock()
				if remaining <= 0 {
					next := curStage + 1
					if next < len(musicStageDurations) {
						m.startStageTimer(next)
					} else {
						m.timeoutRound()
					}
					return
				}
				m.broadcast()
			}
		}
	}()
}

func (m *MusicQuizManager) timeoutRound() {
	m.mu.Lock()
	if m.currentIdx < len(m.questions) {
		cur := m.questions[m.currentIdx]
		m.revealTheme = &MusicRevealData{
			AnimeName: cur.AnimeName,
			SongTitle: cur.SongTitle,
			Type:      cur.Type,
			Sequence:  cur.Sequence,
		}
	}
	m.mu.Unlock()
	m.broadcast()
	time.AfterFunc(musicRevealPauseMs*time.Millisecond, m.advanceToNext)
}

func (m *MusicQuizManager) advanceToNext() {
	m.stopTimer()
	m.mu.Lock()
	m.revealTheme = nil
	m.roundAnswered = map[string]bool{}
	m.roundWinner = ""
	m.roundScored = []QuizScoredEntry{}
	total := len(m.questions)
	m.currentIdx++
	if m.currentIdx >= total {
		m.phase = "results"
		m.mu.Unlock()
		m.broadcast()
		return
	}
	m.mu.Unlock()
	m.startStageTimer(0)
}

// ─── Public actions ───────────────────────────────────────────────────────────

// animeFranchiseKey normalises an anime name to a franchise root so that
// variants of the same series collapse to the same key, e.g.:
//   "One Piece" / "One Piece Film: Red" / "One Piece: Stampede" → "one piece"
//   "Dragon Ball" / "Dragon Ball Z" / "Dragon Ball GT" / "Dragon Ball Super" → "dragon ball"
//   "Sword Art Online" / "Sword Art Online II" / "Sword Art Online: Alicization" → "sword art"
//
// Rules (applied in order):
//  1. Strip known film/movie/special/ova markers (e.g. " film", " movie").
//  2. Strip subtitle introduced by ": " when the base is ≥ 6 chars
//     (avoids false positive on "Re:Zero" whose colon is at position 2).
//  3. For names that are still 3+ words, keep only the first two words — this
//     collapses sequel suffixes like " Z", " GT", " Super", " II", " Season 2".
func animeFranchiseKey(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))

	for _, marker := range []string{" film", " movie", " the movie", " special", " ova"} {
		if i := strings.Index(s, marker); i > 2 {
			s = strings.TrimSpace(s[:i])
			break
		}
	}

	if i := strings.Index(s, ": "); i >= 6 {
		s = strings.TrimSpace(s[:i])
	}

	// Collapse sequel-word suffixes: "Dragon Ball Z/GT/Super" → "dragon ball"
	if words := strings.Fields(s); len(words) >= 3 {
		return words[0] + " " + words[1]
	}
	return s
}

// pickUniqueAnimeThemes implements "anime-first" selection:
//  1. Group all themes by franchise key (animeFranchiseKey).
//  2. Shuffle the list of unique franchises randomly.
//  3. For each franchise pick one random theme — guaranteeing NO franchise
//     repeats regardless of how many tracks it has in the pool.
//
// If there are fewer franchises than n, falls back to filling remaining slots
// with any unused theme IDs (never the same track twice).
func pickUniqueAnimeThemes(themes []MusicQuizTheme, n int) []MusicQuizTheme {
	// 1. Group by franchise
	byFranchise := make(map[string][]MusicQuizTheme, 256)
	for i := range themes {
		key := animeFranchiseKey(themes[i].AnimeName)
		byFranchise[key] = append(byFranchise[key], themes[i])
	}

	// 2. Shuffle franchise keys
	keys := make([]string, 0, len(byFranchise))
	for k := range byFranchise {
		keys = append(keys, k)
	}
	rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })

	// 3. Pick one random theme per franchise
	result := make([]MusicQuizTheme, 0, n)
	for _, k := range keys {
		if len(result) >= n {
			break
		}
		group := byFranchise[k]
		result = append(result, group[rand.Intn(len(group))])
	}

	// Fallback: fill remaining slots if pool has too few franchises
	if len(result) < n {
		seenID := make(map[int]bool, len(result))
		for _, t := range result {
			seenID[t.ID] = true
		}
		for _, group := range byFranchise {
			for _, t := range group {
				if len(result) >= n {
					break
				}
				if !seenID[t.ID] {
					seenID[t.ID] = true
					result = append(result, t)
				}
			}
		}
	}
	return result
}

func (m *MusicQuizManager) OpenLobby(settings MusicQuizSettings, players []QuizPlayer) error {
	// Validate & clamp settings
	if settings.Rounds <= 0 || settings.Rounds > 20 {
		settings.Rounds = 10
	}

	// Select questions from server-side pool
	m.poolMu.RLock()
	ready    := m.poolReady
	pool     := m.themePool
	popular  := m.poolPopular
	poolErr  := m.poolError
	m.poolMu.RUnlock()

	if !ready {
		if poolErr != "" {
			return fmt.Errorf("пул треков недоступен: %s", poolErr)
		}
		return fmt.Errorf("пул треков ещё загружается, попробуйте через несколько секунд")
	}

	// Use the entire pool so every franchise has an equal chance of selection.
	// The anime-first grouping in pickUniqueAnimeThemes already guarantees variety.
	// The popular/MAL sort is still preserved inside the pool for fallback ordering.
	_ = popular // kept in snapshot for UI display

	// Filter by allowed types
	filtered := pool
	if len(settings.AllowedTypes) > 0 {
		typeSet := make(map[string]bool)
		for _, t := range settings.AllowedTypes {
			typeSet[strings.ToUpper(t)] = true
		}
		filtered = nil
		for _, th := range pool {
			if typeSet[strings.ToUpper(th.Type)] {
				filtered = append(filtered, th)
			}
		}
	}
	if len(filtered) == 0 {
		return fmt.Errorf("нет треков с выбранными типами")
	}

	// Shuffle, deduplicate by anime, then take rounds
	questions := pickUniqueAnimeThemes(filtered, settings.Rounds)

	m.stopTimer()
	m.mu.Lock()
	m.phase = "lobby"
	m.questions = questions
	m.settings = settings
	m.players = players
	m.scores = map[string]int{}
	for _, p := range players {
		m.scores[p.ID] = 0
	}
	m.currentIdx = 0
	m.roundAnswered = map[string]bool{}
	m.roundWinner = ""
	m.revealTheme = nil
	m.roundScored = []QuizScoredEntry{}
	m.stopCh = nil
	m.mu.Unlock()
	m.broadcast()
	return nil
}

func (m *MusicQuizManager) JoinLobby(player QuizPlayer) {
	m.mu.Lock()
	if m.phase != "lobby" {
		m.mu.Unlock()
		return
	}
	for _, p := range m.players {
		if p.ID == player.ID {
			m.mu.Unlock()
			return
		}
	}
	m.players = append(m.players, player)
	if _, ok := m.scores[player.ID]; !ok {
		m.scores[player.ID] = 0
	}
	m.mu.Unlock()
	m.broadcast()
}

func (m *MusicQuizManager) StartGame() {
	m.mu.Lock()
	if m.phase != "lobby" {
		m.mu.Unlock()
		return
	}
	m.phase = "playing"
	m.currentIdx = 0
	m.roundAnswered = map[string]bool{}
	m.roundWinner = ""
	m.revealTheme = nil
	m.roundScored = []QuizScoredEntry{}
	m.mu.Unlock()
	m.startStageTimer(0)
}

func (m *MusicQuizManager) SubmitAnswer(playerID, playerName, text string) bool {
	m.mu.Lock()
	if m.phase != "playing" || m.roundAnswered[playerID] || m.currentIdx >= len(m.questions) {
		m.mu.Unlock()
		return false
	}
	cur := m.questions[m.currentIdx]
	if !quizIsMatch(text, QuizAnime{Name: cur.AnimeName, Russian: cur.AnimeName}) {
		m.mu.Unlock()
		return false
	}
	isFirst := m.roundWinner == ""
	points := 1
	if isFirst {
		points = 3
		m.roundWinner = playerID
	}
	found := false
	for _, p := range m.players {
		if p.ID == playerID {
			found = true
			break
		}
	}
	if !found {
		m.players = append(m.players, QuizPlayer{ID: playerID, Name: playerName})
	}
	m.scores[playerID] += points
	m.roundAnswered[playerID] = true
	m.roundScored = append(m.roundScored, QuizScoredEntry{
		PlayerID: playerID, PlayerName: playerName, Points: points, IsFirst: isFirst,
	})
	allDone := len(m.players) > 0
	for _, p := range m.players {
		if !m.roundAnswered[p.ID] {
			allDone = false
			break
		}
	}
	m.mu.Unlock()
	m.broadcast()
	if allDone {
		m.stopTimer()
		time.AfterFunc(2*time.Second, m.advanceToNext)
	}
	return true
}

func (m *MusicQuizManager) Stop() {
	m.stopTimer()
	m.mu.Lock()
	m.phase = "idle"
	m.players = nil
	m.scores = map[string]int{}
	m.questions = nil
	m.settings = defaultMusicSettings()
	m.currentIdx = 0
	m.roundAnswered = map[string]bool{}
	m.roundWinner = ""
	m.revealTheme = nil
	m.roundScored = []QuizScoredEntry{}
	m.mu.Unlock()
	m.broadcast()
}

func (m *MusicQuizManager) PlayAgain() {
	// Re-select questions from pool with same settings
	m.mu.Lock()
	settings := m.settings
	m.mu.Unlock()

	m.poolMu.RLock()
	pool := m.themePool
	m.poolMu.RUnlock()

	filtered := pool
	if len(settings.AllowedTypes) > 0 {
		typeSet := make(map[string]bool)
		for _, t := range settings.AllowedTypes {
			typeSet[strings.ToUpper(t)] = true
		}
		filtered = nil
		for _, th := range pool {
			if typeSet[strings.ToUpper(th.Type)] {
				filtered = append(filtered, th)
			}
		}
	}

	questions := pickUniqueAnimeThemes(filtered, settings.Rounds)

	m.stopTimer()
	m.mu.Lock()
	m.phase = "lobby"
	m.questions = questions
	m.currentIdx = 0
	m.roundAnswered = map[string]bool{}
	m.roundWinner = ""
	m.revealTheme = nil
	m.roundScored = []QuizScoredEntry{}
	m.scores = map[string]int{}
	for _, p := range m.players {
		m.scores[p.ID] = 0
	}
	m.mu.Unlock()
	m.broadcast()
}

// ─── HTTP handlers ────────────────────────────────────────────────────────────

func (m *MusicQuizManager) ServeState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.snapshot())
}

func (m *MusicQuizManager) ServeOpenLobby(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Settings MusicQuizSettings `json:"settings"`
		Players  []QuizPlayer      `json:"players"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := m.OpenLobby(body.Settings, body.Players); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (m *MusicQuizManager) ServeJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var p QuizPlayer
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.ID == "" {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	m.JoinLobby(p)
	w.WriteHeader(http.StatusOK)
}

func (m *MusicQuizManager) ServeStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	m.StartGame()
	w.WriteHeader(http.StatusOK)
}

func (m *MusicQuizManager) ServeAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		PlayerID   string `json:"playerId"`
		PlayerName string `json:"playerName"`
		Text       string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PlayerID == "" {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	correct := m.SubmitAnswer(body.PlayerID, body.PlayerName, body.Text)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"correct": correct})
}

func (m *MusicQuizManager) ServeStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	m.Stop()
	w.WriteHeader(http.StatusOK)
}

func (m *MusicQuizManager) ServeAgain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	m.PlayAgain()
	w.WriteHeader(http.StatusOK)
}

// ServeReloadPool resets a previously failed pool and starts a fresh background
// fetch. Silently ignored if the pool is already ready or currently loading.
func (m *MusicQuizManager) ServeReloadPool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	m.poolMu.Lock()
	hasError := m.poolError != ""
	if hasError {
		m.poolError = ""
		m.themePool = nil
		m.poolReady = false
	}
	m.poolMu.Unlock()
	if hasError {
		m.broadcast() // switch clients back to "loading" state
		go m.fetchPool()
	}
	w.WriteHeader(http.StatusOK)
}

// ServeAnimeNames returns a deduplicated, sorted list of anime names from the
// theme pool. Used by the frontend for autocomplete suggestions in the music quiz.
// Returns "no-store" cache headers when the pool is not yet ready so that
// clients don't cache an empty or partial list.
func (m *MusicQuizManager) ServeAnimeNames(w http.ResponseWriter, r *http.Request) {
	m.poolMu.RLock()
	pool  := m.themePool
	ready := m.poolReady
	m.poolMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if !ready {
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode([]string{})
		return
	}

	seen := make(map[string]struct{}, len(pool))
	for _, t := range pool {
		seen[t.AnimeName] = struct{}{}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(names)
}
