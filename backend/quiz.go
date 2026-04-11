package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
)

// ─── Config ───────────────────────────────────────────────────────────────────

var animeStageDurations = []int{12, 10, 8, 8}

const animeRevealPauseMs = 4500

// ─── Settings ─────────────────────────────────────────────────────────────────

type AnimeQuizSettings struct {
	Rounds int `json:"rounds"` // 1-20
}

func defaultAnimeSettings() AnimeQuizSettings {
	return AnimeQuizSettings{Rounds: 10}
}

// ─── Types ────────────────────────────────────────────────────────────────────

type QuizPlayer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// QuizAnime holds full question data — names are kept server-side and never
// sent to clients while the round is active (only on reveal).
type QuizAnime struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Russian string `json:"russian"`
	Poster  string `json:"poster"`
}

type QuizScoredEntry struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Points     int    `json:"points"`
	IsFirst    bool   `json:"isFirst"`
}

// QuizState is broadcast to every connected client on every change.
// Anime names are intentionally omitted from the current question to prevent spoilers.
type QuizState struct {
	Phase         string            `json:"phase"`
	Players       []QuizPlayer      `json:"players"`
	Scores        map[string]int    `json:"scores"`
	CurrentIdx    int               `json:"currentIdx"`
	TotalQ        int               `json:"totalQ"`
	Stage         int               `json:"stage"`
	StageTimeLeft int               `json:"stageTimeLeft"`
	RoundAnswered map[string]bool   `json:"roundAnswered"`
	RoundWinner   string            `json:"roundWinner"`
	RevealAnime   *QuizAnime        `json:"revealAnime"` // non-nil during reveal (contains names)
	RoundScored   []QuizScoredEntry `json:"roundScored"`
	// Current question — only ID + poster (no names until reveal)
	AnimeID     int               `json:"animeId"`
	AnimePoster string            `json:"animePoster"`
	MaskedTitle string            `json:"maskedTitle"` // server-computed letter-masked hint
	Settings    AnimeQuizSettings `json:"settings"`
}

// ─── Manager ──────────────────────────────────────────────────────────────────

type QuizManager struct {
	hub *Hub

	mu            sync.Mutex
	phase         string
	players       []QuizPlayer
	scores        map[string]int
	questions     []QuizAnime // full list with names — never broadcast as-is
	settings      AnimeQuizSettings
	currentIdx    int
	stage         int
	stageTimeLeft int
	roundAnswered map[string]bool
	roundWinner   string
	revealAnime   *QuizAnime
	roundScored   []QuizScoredEntry
	stopCh        chan struct{}
}

func newQuizManager(hub *Hub) *QuizManager {
	return &QuizManager{
		hub:           hub,
		phase:         "idle",
		settings:      defaultAnimeSettings(),
		scores:        map[string]int{},
		roundAnswered: map[string]bool{},
		roundScored:   []QuizScoredEntry{},
	}
}

// ─── State helpers ────────────────────────────────────────────────────────────

func (q *QuizManager) snapshot() QuizState {
	q.mu.Lock()
	defer q.mu.Unlock()

	ps := make([]QuizPlayer, len(q.players))
	copy(ps, q.players)
	sc := make(map[string]int, len(q.scores))
	for k, v := range q.scores {
		sc[k] = v
	}
	ra := make(map[string]bool, len(q.roundAnswered))
	for k, v := range q.roundAnswered {
		ra[k] = v
	}
	scored := make([]QuizScoredEntry, len(q.roundScored))
	copy(scored, q.roundScored)

	state := QuizState{
		Phase:         q.phase,
		Players:       ps,
		Scores:        sc,
		CurrentIdx:    q.currentIdx,
		Stage:         q.stage,
		StageTimeLeft: q.stageTimeLeft,
		RoundAnswered: ra,
		RoundWinner:   q.roundWinner,
		RevealAnime:   q.revealAnime,
		RoundScored:   scored,
	}
	state.Settings = q.settings
	if len(q.questions) > 0 {
		state.TotalQ = minInt(q.settings.Rounds, len(q.questions))
		if q.currentIdx < len(q.questions) {
			cur := q.questions[q.currentIdx]
			state.AnimeID = cur.ID
			state.AnimePoster = cur.Poster
			// Compute masked title server-side so clients can't cheat
			title := cur.Russian
			if title == "" {
				title = cur.Name
			}
			state.MaskedTitle = maskedTitle(title, cur.ID, q.stage)
		}
	}
	return state
}

func (q *QuizManager) broadcast() {
	state := q.snapshot()
	payload, _ := json.Marshal(state)
	q.hub.BroadcastAll("quiz-state", payload)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Timer ────────────────────────────────────────────────────────────────────

func (q *QuizManager) stopTimer() {
	q.mu.Lock()
	if q.stopCh != nil {
		select {
		case <-q.stopCh:
		default:
			close(q.stopCh)
		}
		q.stopCh = nil
	}
	q.mu.Unlock()
}

func (q *QuizManager) startStageTimer(s int) {
	q.stopTimer()

	q.mu.Lock()
	q.stage = s
	q.stageTimeLeft = animeStageDurations[s]
	stopCh := make(chan struct{})
	q.stopCh = stopCh
	q.mu.Unlock()

	q.broadcast()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				q.mu.Lock()
				q.stageTimeLeft--
				remaining := q.stageTimeLeft
				curStage := q.stage
				q.mu.Unlock()
				if remaining <= 0 {
					next := curStage + 1
					if next < len(animeStageDurations) {
						q.startStageTimer(next)
					} else {
						q.timeoutRound()
					}
					return
				}
				q.broadcast()
			}
		}
	}()
}

func (q *QuizManager) timeoutRound() {
	q.mu.Lock()
	if q.currentIdx < len(q.questions) {
		c := q.questions[q.currentIdx]
		q.revealAnime = &c
	}
	q.mu.Unlock()
	q.broadcast()
	time.AfterFunc(animeRevealPauseMs*time.Millisecond, q.advanceToNext)
}

func (q *QuizManager) advanceToNext() {
	q.stopTimer()
	q.mu.Lock()
	q.revealAnime = nil
	q.roundAnswered = map[string]bool{}
	q.roundWinner = ""
	q.roundScored = []QuizScoredEntry{}
	total := minInt(q.settings.Rounds, len(q.questions))
	q.currentIdx++
	if q.currentIdx >= total {
		q.phase = "results"
		q.mu.Unlock()
		q.broadcast()
		return
	}
	q.mu.Unlock()
	q.startStageTimer(0)
}

// ─── Title masking (mirrors JS getMaskedTitle) ────────────────────────────────

func quizSeededShuffle(n, seed int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	s := seed
	if s < 0 {
		s = -s
	}
	s = s & 0x7fffffff
	for i := n - 1; i > 0; i-- {
		s = int((int64(s)*1664525 + 1013904223) & 0x7fffffff)
		j := s % (i + 1)
		idx[i], idx[j] = idx[j], idx[i]
	}
	return idx
}

func maskedTitle(title string, animeID, stageIndex int) string {
	chars := []rune(title)
	var letterPos []int
	for i, c := range chars {
		if unicode.IsLetter(c) {
			letterPos = append(letterPos, i)
		}
	}
	if len(letterPos) == 0 {
		return title
	}
	const totalStages = 4
	shuffled := quizSeededShuffle(len(letterPos), animeID)
	fraction := float64(stageIndex) / float64(totalStages)
	revealCnt := int(fraction * float64(len(letterPos)))

	revealSet := make(map[int]bool, revealCnt)
	for _, idx := range shuffled[:revealCnt] {
		revealSet[letterPos[idx]] = true
	}
	result := make([]rune, len(chars))
	for i, c := range chars {
		if unicode.IsLetter(c) {
			if revealSet[i] {
				result[i] = c
			} else {
				result[i] = '_'
			}
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// ─── Answer validation ────────────────────────────────────────────────────────

func quizNorm(s string) string {
	s = strings.ToLower(s)
	var out strings.Builder
	prev := ' '
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			prev = r
		} else if prev != ' ' {
			out.WriteRune(' ')
			prev = ' '
		}
	}
	return strings.TrimSpace(out.String())
}

func quizLevenshtein(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	row := make([]int, len(b)+1)
	for i := range row {
		row[i] = i
	}
	for i := 0; i < len(a); i++ {
		prev := row[0]
		row[0] = i + 1
		for j := 0; j < len(b); j++ {
			tmp := row[j+1]
			if a[i] == b[j] {
				row[j+1] = prev
			} else {
				m := prev
				if row[j] < m {
					m = row[j]
				}
				if row[j+1] < m {
					m = row[j+1]
				}
				row[j+1] = 1 + m
			}
			prev = tmp
		}
	}
	return row[len(b)]
}

func quizIsMatch(input string, anime QuizAnime) bool {
	inp := quizNorm(input)
	if len([]rune(inp)) < 2 {
		return false
	}
	inpR := []rune(inp)
	n := len(inpR)
	maxDist := 0
	switch {
	case n > 13:
		maxDist = 3
	case n > 8:
		maxDist = 2
	case n > 4:
		maxDist = 1
	}
	targets := []string{quizNorm(anime.Russian), quizNorm(anime.Name)}
	for _, t := range targets {
		if t == "" {
			continue
		}
		tr := []rune(t)
		if t == inp {
			return true
		}
		if n >= 4 && strings.HasPrefix(t, inp) {
			return true
		}
		if maxDist > 0 {
			if quizLevenshtein(tr, inpR) <= maxDist {
				return true
			}
			if n >= 4 && len(tr) >= n && quizLevenshtein(tr[:n], inpR) <= maxDist {
				return true
			}
		}
	}
	return false
}

// ─── Public actions ───────────────────────────────────────────────────────────

func (q *QuizManager) OpenLobby(settings AnimeQuizSettings, questions []QuizAnime, players []QuizPlayer) {
	if settings.Rounds <= 0 || settings.Rounds > 20 {
		settings.Rounds = 10
	}
	// Trim questions list to requested rounds count
	if len(questions) > settings.Rounds {
		questions = questions[:settings.Rounds]
	}
	q.stopTimer()
	q.mu.Lock()
	q.phase = "lobby"
	q.settings = settings
	q.questions = questions
	q.players = players
	q.scores = map[string]int{}
	for _, p := range players {
		q.scores[p.ID] = 0
	}
	q.currentIdx = 0
	q.roundAnswered = map[string]bool{}
	q.roundWinner = ""
	q.revealAnime = nil
	q.roundScored = []QuizScoredEntry{}
	q.stopCh = nil
	q.mu.Unlock()
	q.broadcast()
}

func (q *QuizManager) JoinLobby(player QuizPlayer) {
	q.mu.Lock()
	if q.phase != "lobby" {
		q.mu.Unlock()
		return
	}
	for _, p := range q.players {
		if p.ID == player.ID {
			q.mu.Unlock()
			return
		}
	}
	q.players = append(q.players, player)
	if _, ok := q.scores[player.ID]; !ok {
		q.scores[player.ID] = 0
	}
	q.mu.Unlock()
	q.broadcast()
}

func (q *QuizManager) StartGame() {
	q.mu.Lock()
	if q.phase != "lobby" {
		q.mu.Unlock()
		return
	}
	qs := make([]QuizAnime, len(q.questions))
	copy(qs, q.questions)
	rand.Shuffle(len(qs), func(i, j int) { qs[i], qs[j] = qs[j], qs[i] })
	q.questions = qs
	q.phase = "playing"
	q.currentIdx = 0
	q.roundAnswered = map[string]bool{}
	q.roundWinner = ""
	q.revealAnime = nil
	q.roundScored = []QuizScoredEntry{}
	q.mu.Unlock()
	q.startStageTimer(0)
}

func (q *QuizManager) SubmitAnswer(playerID, playerName, text string) bool {
	q.mu.Lock()
	if q.phase != "playing" || q.roundAnswered[playerID] || q.currentIdx >= len(q.questions) {
		q.mu.Unlock()
		return false
	}
	cur := q.questions[q.currentIdx]
	if !quizIsMatch(text, cur) {
		q.mu.Unlock()
		return false
	}
	isFirst := q.roundWinner == ""
	points := 1
	if isFirst {
		points = 3
		q.roundWinner = playerID
	}
	found := false
	for _, p := range q.players {
		if p.ID == playerID {
			found = true
			break
		}
	}
	if !found {
		q.players = append(q.players, QuizPlayer{ID: playerID, Name: playerName})
	}
	q.scores[playerID] += points
	q.roundAnswered[playerID] = true
	q.roundScored = append(q.roundScored, QuizScoredEntry{
		PlayerID: playerID, PlayerName: playerName, Points: points, IsFirst: isFirst,
	})
	allDone := len(q.players) > 0
	for _, p := range q.players {
		if !q.roundAnswered[p.ID] {
			allDone = false
			break
		}
	}
	q.mu.Unlock()
	q.broadcast()
	if allDone {
		q.stopTimer()
		time.AfterFunc(2*time.Second, q.advanceToNext)
	}
	return true
}

func (q *QuizManager) Stop() {
	q.stopTimer()
	q.mu.Lock()
	q.phase = "idle"
	q.players = nil
	q.scores = map[string]int{}
	q.questions = nil
	q.currentIdx = 0
	q.roundAnswered = map[string]bool{}
	q.roundWinner = ""
	q.revealAnime = nil
	q.roundScored = []QuizScoredEntry{}
	q.mu.Unlock()
	q.broadcast()
}

func (q *QuizManager) PlayAgain() {
	q.stopTimer()
	q.mu.Lock()
	q.phase = "lobby"
	q.currentIdx = 0
	q.roundAnswered = map[string]bool{}
	q.roundWinner = ""
	q.revealAnime = nil
	q.roundScored = []QuizScoredEntry{}
	q.scores = map[string]int{}
	for _, p := range q.players {
		q.scores[p.ID] = 0
	}
	q.mu.Unlock()
	q.broadcast()
}

// ─── HTTP handlers ────────────────────────────────────────────────────────────

func (q *QuizManager) ServeState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q.snapshot())
}

func (q *QuizManager) ServeOpenLobby(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Settings  AnimeQuizSettings `json:"settings"`
		Questions []QuizAnime       `json:"questions"`
		Players   []QuizPlayer      `json:"players"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Questions) == 0 {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	q.OpenLobby(body.Settings, body.Questions, body.Players)
	w.WriteHeader(http.StatusOK)
}

func (q *QuizManager) ServeJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var p QuizPlayer
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.ID == "" {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	q.JoinLobby(p)
	w.WriteHeader(http.StatusOK)
}

func (q *QuizManager) ServeStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	q.StartGame()
	w.WriteHeader(http.StatusOK)
}

func (q *QuizManager) ServeAnswer(w http.ResponseWriter, r *http.Request) {
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
	ok := q.SubmitAnswer(body.PlayerID, body.PlayerName, body.Text)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"correct": ok})
}

func (q *QuizManager) ServeStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	q.Stop()
	w.WriteHeader(http.StatusOK)
}

func (q *QuizManager) ServeAgain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	q.PlayAgain()
	w.WriteHeader(http.StatusOK)
}
