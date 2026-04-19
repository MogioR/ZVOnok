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

	rooms := newRoomManager()

	// roomFromReq returns the room indicated by the optional ?room= query param.
	// Defaults to the default room when the param is absent or unknown.
	roomFromReq := func(r *http.Request) *Room {
		id := r.URL.Query().Get("room")
		if id == "" {
			id = DefaultRoomID
		}
		room := rooms.Get(id)
		if room == nil {
			room = rooms.Get(DefaultRoomID)
		}
		return room
	}

	// ─── WebSocket ──────────────────────────────────────────────────────────────
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("room")
		if roomID == "" {
			roomID = DefaultRoomID
		}
		room := rooms.Get(roomID)
		if room == nil {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}
		serveWs(room.hub, w, r)
	})

	// ─── ICE servers ────────────────────────────────────────────────────────────
	http.HandleFunc("/ice-servers", handleIceServers)

	// ─── Room management API ────────────────────────────────────────────────────

	// GET /api/rooms/info?room=<id>  → {exists, hasPassword, roomId}
	http.HandleFunc("/api/rooms/info", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("room")
		info := rooms.Info(id)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(info)
	})

	// POST /api/rooms/create  body: {roomId, password}  → {roomId, hasPassword}
	http.HandleFunc("/api/rooms/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			RoomID   string `json:"roomId"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RoomID == "" {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		room := rooms.Create(body.RoomID, body.Password)
		if room == nil {
			http.Error(w, "room already exists or invalid ID", http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"roomId":      room.ID,
			"hasPassword": room.hub.password != "",
		})
	})

	// ─── Anime data proxies (Shikimori) ─────────────────────────────────────────
	http.HandleFunc("/api/quiz/animes", handleQuizAnimes)
	http.HandleFunc("/api/quiz/screenshots", handleQuizScreenshots)
	// Music themes + audio proxy (AnimeThemes.moe)
	http.HandleFunc("/api/quiz/music/themes", handleQuizMusicThemes)
	http.HandleFunc("/api/quiz/music/audio", handleQuizMusicAudio)

	// ─── Anime quiz game API (per-room) ─────────────────────────────────────────
	http.HandleFunc("/api/quiz/state", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).quiz.ServeState(w, r)
	})
	http.HandleFunc("/api/quiz/lobby", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).quiz.ServeOpenLobby(w, r)
	})
	http.HandleFunc("/api/quiz/join", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).quiz.ServeJoin(w, r)
	})
	http.HandleFunc("/api/quiz/start", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).quiz.ServeStart(w, r)
	})
	http.HandleFunc("/api/quiz/answer", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).quiz.ServeAnswer(w, r)
	})
	http.HandleFunc("/api/quiz/stop", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).quiz.ServeStop(w, r)
	})
	http.HandleFunc("/api/quiz/again", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).quiz.ServeAgain(w, r)
	})

	// ─── Music quiz game API (per-room) ─────────────────────────────────────────
	http.HandleFunc("/api/musicquiz/animenames", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).musicQuiz.ServeAnimeNames(w, r)
	})
	http.HandleFunc("/api/musicquiz/reload-pool", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).musicQuiz.ServeReloadPool(w, r)
	})
	http.HandleFunc("/api/musicquiz/state", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).musicQuiz.ServeState(w, r)
	})
	http.HandleFunc("/api/musicquiz/lobby", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).musicQuiz.ServeOpenLobby(w, r)
	})
	http.HandleFunc("/api/musicquiz/join", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).musicQuiz.ServeJoin(w, r)
	})
	http.HandleFunc("/api/musicquiz/start", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).musicQuiz.ServeStart(w, r)
	})
	http.HandleFunc("/api/musicquiz/answer", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).musicQuiz.ServeAnswer(w, r)
	})
	http.HandleFunc("/api/musicquiz/stop", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).musicQuiz.ServeStop(w, r)
	})
	http.HandleFunc("/api/musicquiz/again", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).musicQuiz.ServeAgain(w, r)
	})

	// ─── Music bot API (per-room) ────────────────────────────────────────────────
	http.HandleFunc("/api/music/stream", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).music.ServeStream(w, r)
	})
	http.HandleFunc("/api/music/state", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).music.ServeState(w, r)
	})
	http.HandleFunc("/api/music/search", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).music.ServeSearch(w, r)
	})
	http.HandleFunc("/api/music/add", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).music.ServeAdd(w, r)
	})
	http.HandleFunc("/api/music/add-playlist", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).music.ServeAddPlaylist(w, r)
	})
	http.HandleFunc("/api/music/remove", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).music.ServeRemove(w, r)
	})
	http.HandleFunc("/api/music/skip", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).music.ServeSkip(w, r)
	})
	http.HandleFunc("/api/music/clear", func(w http.ResponseWriter, r *http.Request) {
		roomFromReq(r).music.ServeClear(w, r)
	})

	http.Handle("/", http.FileServer(http.Dir(staticDir)))

	log.Printf("ZVOnok server running on http://0.0.0.0:%s (static: %s)", port, staticDir)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleQuizAnimes proxies Shikimori anime list (avoids browser CORS restrictions).
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
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleQuizMusicThemes proxies the AnimeThemes.moe anime-theme list.
func handleQuizMusicThemes(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}

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
