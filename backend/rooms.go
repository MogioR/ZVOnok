package main

import (
	"sync"
	"time"
)

// DefaultRoomID is the ID of the always-present default room.
const DefaultRoomID = "default"

// RoomInfo is what the frontend receives when checking a room.
type RoomInfo struct {
	Exists      bool   `json:"exists"`
	HasPassword bool   `json:"hasPassword"`
	RoomID      string `json:"roomId"`
}

// Room holds all per-room state: its own Hub, MusicManager, QuizManagers.
type Room struct {
	ID        string
	hub       *Hub
	music     *MusicManager
	quiz      *QuizManager
	musicQuiz *MusicQuizManager
	isDefault bool
	createdAt time.Time
}

// RoomManager manages all active rooms.
type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func newRoomManager() *RoomManager {
	rm := &RoomManager{
		rooms: make(map[string]*Room),
	}
	// Create default room — it is never auto-deleted.
	default_ := rm.createRoom(DefaultRoomID, "")
	default_.isDefault = true
	return rm
}

// createRoom instantiates a new Room with its own Hub and managers.
// It is NOT goroutine-safe on its own — callers must hold the write lock or
// call it before the manager is shared.
func (rm *RoomManager) createRoom(id, password string) *Room {
	h := newHub(password)
	go h.run()

	room := &Room{
		ID:        id,
		hub:       h,
		music:     newMusicManager(h),
		quiz:      newQuizManager(h),
		musicQuiz: newMusicQuizManager(h),
		createdAt: time.Now(),
	}

	rm.mu.Lock()
	rm.rooms[id] = room
	rm.mu.Unlock()

	// When the room's hub becomes empty, delete it (unless it is the default).
	h.onEmpty = func() {
		if !room.isDefault {
			rm.mu.Lock()
			delete(rm.rooms, id)
			rm.mu.Unlock()
		}
	}

	return room
}

// Create creates a fresh room with the given ID and password.
// Returns nil if a room with that ID already exists, or the ID is reserved.
func (rm *RoomManager) Create(id, password string) *Room {
	if id == "" || id == DefaultRoomID {
		return nil
	}
	rm.mu.RLock()
	_, exists := rm.rooms[id]
	rm.mu.RUnlock()
	if exists {
		return nil
	}
	return rm.createRoom(id, password)
}

// Get returns the room with the given ID.  Returns nil if not found.
// An empty id is treated as DefaultRoomID.
func (rm *RoomManager) Get(id string) *Room {
	if id == "" {
		id = DefaultRoomID
	}
	rm.mu.RLock()
	r := rm.rooms[id]
	rm.mu.RUnlock()
	return r
}

// Info returns room metadata without exposing internals.
func (rm *RoomManager) Info(id string) RoomInfo {
	if id == "" {
		id = DefaultRoomID
	}
	rm.mu.RLock()
	r, ok := rm.rooms[id]
	rm.mu.RUnlock()
	if !ok {
		return RoomInfo{Exists: false, RoomID: id}
	}
	return RoomInfo{
		Exists:      true,
		HasPassword: r.hub.password != "",
		RoomID:      id,
	}
}
