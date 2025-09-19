package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
	ctx   context.Context
}

func NewManager(ctx context.Context) *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
		ctx:   ctx,
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	roomCode := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("room")))
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if roomCode == "" && name == "" {
		http.Error(w, "Rooom and Username is required", http.StatusBadRequest)
		return
	}
	conn, err := websocketUpgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}
	room := m.GetOrCreate(roomCode)
	client := NewClient(conn, name, room)
	log.Printf("Connected to Client %s\n", name)
	room.register <- client
	go client.writePump()
	go client.readPump()
}

func (m *Manager) GetOrCreate(code string) *Room {
	code = strings.ToUpper(strings.TrimSpace(code))

	m.mu.RLock()
	room, ok := m.rooms[code]
	m.mu.RUnlock()
	if ok {
		return room
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if room, ok = m.rooms[code]; ok {
		return room
	}

	room = NewRoom(code)
	m.rooms[code] = room
	go room.run()
	return room
}

func (m *Manager) GetRooms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	m.mu.RLock()
	items := make([]roomItem, 0, len(m.rooms))
	for code, room := range m.rooms {
		room.mu.RLock()
		cnt := len(room.Clients)
		room.mu.RUnlock()

		items = append(items, roomItem{
			ID:    code,
			Name:  code,
			Code:  code,
			Count: cnt,
		})
	}
	m.mu.RUnlock()

	if err := json.NewEncoder(w).Encode(items); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
