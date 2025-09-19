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
	CheckOrigin:     checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	switch origin {
	case "http://localhost:3000":
		return true
	default:
		return false
	}
}

type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
	ctx   context.Context
}

func (m *Manager) getRooms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type roomItem struct {
		ID    string `json:"id"`   // FRONT: value.split('|') uchun
		Name  string `json:"name"` // ko‘rinadigan nom
		Code  string `json:"code"` // ixtiyoriy, foydali
		Count int    `json:"count"`
	}

	m.mu.RLock()
	items := make([]roomItem, 0, len(m.rooms))
	for code, room := range m.rooms {
		room.mu.RLock()
		cnt := len(room.Clients)
		room.mu.RUnlock()

		name := code
		items = append(items, roomItem{
			ID:    code + "|" + name,
			Name:  name,
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

func (m *Manager) GetOrCreate(code string) *Room {
	code = strings.ToUpper(strings.TrimSpace(code))

	m.mu.RLock()
	r := m.rooms[code]
	m.mu.RUnlock()
	if r != nil {
		return r
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if r = m.rooms[code]; r != nil {
		return r
	}
	r = NewRoom(code)
	m.rooms[code] = r
	go r.run()
	return r
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

	if roomCode == "" || name == "" {
		http.Error(w, "room & name required", http.StatusBadRequest)
		return
	}

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	log.Printf("WS joined room=%s name=%s", roomCode, name)

	room := m.GetOrCreate(roomCode)
	client := NewClient(conn, room, name)

	room.register <- client

	go client.writePump()
	go client.readPump()
}
