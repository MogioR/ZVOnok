package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Participant holds public user info broadcast to peers.
type Participant struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// Client represents one connected WebSocket user.
type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	Participant Participant
	joined      bool
}

// Hub maintains all active clients and routes messages between them.
type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	forward    chan *forwardMsg
}

type forwardMsg struct {
	from    string
	to      string // empty = broadcast to all except from
	msgType string
	payload json.RawMessage
}

// WSMessage is the common JSON envelope for all WebSocket messages.
type WSMessage struct {
	Type    string          `json:"type"`
	From    string          `json:"from,omitempty"`
	To      string          `json:"to,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		forward:    make(chan *forwardMsg, 512),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			existing := make([]Participant, 0, len(h.clients))
			for _, c := range h.clients {
				if c.joined {
					existing = append(existing, c.Participant)
				}
			}
			h.clients[client.Participant.ID] = client
			h.mu.Unlock()

			welcomePayload, _ := json.Marshal(map[string]interface{}{
				"id":           client.Participant.ID,
				"participants": existing,
			})
			h.sendTo(client, WSMessage{Type: "welcome", Payload: welcomePayload})

			joinPayload, _ := json.Marshal(client.Participant)
			h.broadcastExcept(client.Participant.ID, WSMessage{
				Type:    "peer-joined",
				From:    client.Participant.ID,
				Payload: joinPayload,
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.Participant.ID]; ok {
				delete(h.clients, client.Participant.ID)
				close(client.send)
				h.mu.Unlock()

				if client.joined {
					leavePayload, _ := json.Marshal(map[string]string{"id": client.Participant.ID})
					h.broadcastExcept(client.Participant.ID, WSMessage{
						Type:    "peer-left",
						From:    client.Participant.ID,
						Payload: leavePayload,
					})
				}
			} else {
				h.mu.Unlock()
			}

		case msg := <-h.forward:
			if msg.to != "" {
				h.mu.RLock()
				target, ok := h.clients[msg.to]
				h.mu.RUnlock()
				if ok {
					outMsg, _ := json.Marshal(WSMessage{
						Type:    msg.msgType,
						From:    msg.from,
						To:      msg.to,
						Payload: msg.payload,
					})
					select {
					case target.send <- outMsg:
					default:
						log.Printf("send buffer full for client %s", msg.to)
					}
				}
			} else {
				outMsg, _ := json.Marshal(WSMessage{
					Type:    msg.msgType,
					From:    msg.from,
					Payload: msg.payload,
				})
				h.mu.RLock()
				for id, c := range h.clients {
					if id != msg.from {
						select {
						case c.send <- outMsg:
						default:
						}
					}
				}
				h.mu.RUnlock()
			}
		}
	}
}

func (h *Hub) sendTo(client *Client, msg WSMessage) {
	data, _ := json.Marshal(msg)
	select {
	case client.send <- data:
	default:
		log.Printf("send buffer full for client %s", client.Participant.ID)
	}
}

// BroadcastAll sends a server-originated message to every joined client.
func (h *Hub) BroadcastAll(msgType string, payload json.RawMessage) {
	data, _ := json.Marshal(WSMessage{Type: msgType, Payload: payload})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if c.joined {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

func (h *Hub) broadcastExcept(excludeID string, msg WSMessage) {
	data, _ := json.Marshal(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for id, c := range h.clients {
		if id != excludeID {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		Participant: Participant{
			ID: uuid.New().String(),
		},
	}
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("json parse error: %v", err)
			continue
		}

		switch msg.Type {
		case "join":
			var info struct {
				Name   string `json:"name"`
				Avatar string `json:"avatar"`
			}
			if err := json.Unmarshal(msg.Payload, &info); err != nil {
				log.Printf("join parse error: %v", err)
				continue
			}
			c.Participant.Name = info.Name
			c.Participant.Avatar = info.Avatar
			c.joined = true
			c.hub.register <- c

		case "offer", "answer", "ice-candidate":
			if !c.joined {
				continue
			}
			c.hub.forward <- &forwardMsg{
				from:    c.Participant.ID,
				to:      msg.To,
				msgType: msg.Type,
				payload: msg.Payload,
			}

		case "speaking", "screen-share", "muted", "status", "chat", "game":
			if !c.joined {
				continue
			}
			c.hub.forward <- &forwardMsg{
				from:    c.Participant.ID,
				msgType: msg.Type,
				payload: msg.Payload,
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
