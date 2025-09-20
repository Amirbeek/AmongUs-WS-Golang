package main

import (
	"encoding/json"
	"sync"
)

func mustJSON(v interface{}) []byte { b, _ := json.Marshal(v); return b }

type roomItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Code  string `json:"code"`
	Count int    `json:"count"`
}
type Room struct {
	Code       string
	Clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
	Ready      int
	active     int
}

func NewRoom(code string) *Room {
	return &Room{
		Code:       code,
		Clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 128),
		mu:         sync.RWMutex{},
		active:     0,
	}
}

func (r *Room) run() {
	for {
		select {
		case c := <-r.register:
			r.mu.Lock()
			r.Clients[c] = struct{}{}
			r.mu.Unlock()

			//_ = c.conn.WriteJSON(map[string]any{
			//	"type": "hello",
			//	"data": map[string]string{"room": r.Code, "name": c.name},
			//})
			r.active++
			r.broadcastState()
			r.mu.RLock()
			for cl := range r.Clients {
				select {
				case cl.send <- []byte("joined: " + c.name):
				default:
				}
			}
			r.mu.RUnlock()
			r.broadcastState()

		case c := <-r.unregister:
			r.mu.Lock()
			if _, ok := r.Clients[c]; ok {
				delete(r.Clients, c)
				close(c.send)
			}
			r.active--
			r.mu.Unlock()

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

func (r *Room) snapshot() StateOut {
	r.mu.RLock()
	defer r.mu.RUnlock()
	st := StateOut{Room: r.Code}

	for c := range r.Clients {
		st.Players = append(st.Players, PlayerSnap{
			ID:    c.ID,
			Name:  c.name,
			Alive: c.Alive,
			Ready: c.Ready,
		})
	}
	return st
}

func (r *Room) broadcastState() {
	st := r.snapshot()
	env := Envelope{Type: "state", Data: st}
	r.broadcast <- mustJSON(env)
}

func (r *Room) allReady() bool {
	if len(r.Clients) == r.Ready {
		return true
	}
	return false
}

func (r *Room) startGameCountDown() {

}
