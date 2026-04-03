package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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

	music := newMusicManager(hub)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	http.HandleFunc("/ice-servers", handleIceServers)

	// Music API
	http.HandleFunc("/api/music/stream", music.ServeStream)
	http.HandleFunc("/api/music/state", music.ServeState)
	http.HandleFunc("/api/music/search", music.ServeSearch)
	http.HandleFunc("/api/music/add", music.ServeAdd)
	http.HandleFunc("/api/music/remove", music.ServeRemove)
	http.HandleFunc("/api/music/skip", music.ServeSkip)
	http.HandleFunc("/api/music/clear", music.ServeClear)

	http.Handle("/", http.FileServer(http.Dir(staticDir)))

	log.Printf("ZVOnok server running on http://0.0.0.0:%s (static: %s)", port, staticDir)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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
