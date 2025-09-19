package main

import (
	"encoding/json"
	"sync"
)

// JSON ga serialize qilish uchun helper
func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

type roomItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Code  string `json:"code"`
	Count int    `json:"count"`
}

// Room - bitta chat/xona manageri
type Room struct {
	Code       string
	Clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
}

// Yangi xona yaratish
func NewRoom(code string) *Room {
	return &Room{
		Code:       code,
		Clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		mu:         sync.RWMutex{},
	}
}

// Room lifecycle loop
func (r *Room) run() {
	for {
		select {
		// Xonaga yangi client qo‘shildi
		case client := <-r.register:
			r.mu.Lock()
			r.Clients[client] = true
			r.mu.Unlock()

			// Hamma clientlarga joined habar yuborish
			r.mu.RLock()
			for cl := range r.Clients {
				select {
				case cl.send <- []byte("joined: " + cl.name):
				default:
				}
			}
			r.mu.RUnlock()

			r.broadcastState()

		// Client chiqib ketdi
		case client := <-r.unregister:
			r.mu.Lock()
			if _, ok := r.Clients[client]; ok {
				delete(r.Clients, client)
				close(client.send)
			}
			r.mu.Unlock()

		// Broadcast yuborish
		case msg := <-r.broadcast:
			r.mu.RLock()
			for cl := range r.Clients {
				select {
				case cl.send <- msg:
				default:
				}
			}
			r.mu.RUnlock()
		}
	}
}

// Snapshot xolatni qaytaryapmiz
func (r *Room) snapshot() StateOut {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state := StateOut{Room: r.Code}
	for c := range r.Clients {
		state.Players = append(state.Players, PlayerSnap{
			ID:    c.ID,
			Name:  c.name,
			Alive: c.Alive,
			Ready: c.Ready,
		})
	}
	return state
}

// Xona holatini hammaga yuborish
func (r *Room) broadcastState() {
	st := r.snapshot()
	env := Envelope{Type: "state", Data: st}
	r.broadcast <- mustJSON(env)
}
