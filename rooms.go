package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
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
	pubsub     *redis.PubSub
	stop       chan struct{}
	subActive  bool
	subMu      sync.Mutex
}

func NewRoom(code string) *Room {
	return &Room{
		Code:       code,
		Clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 128),
		mu:         sync.RWMutex{},
		stop:       make(chan struct{}),
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
	env := Envelope{Type: "state", Data: r.snapshot()}
	r.Publish(mustJSON(env))
}

func (r *Room) allReady() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Clients) > 0 && len(r.Clients) == r.Ready
}

func (r *Room) startGameCountDown() {

}

func (r *Room) Publish(msg []byte) {
	_ = rdb.Publish(context.Background(), "room:"+r.Code+":pub", msg).Err()
}

func (r *Room) ensureSubscriber() {
	r.subMu.Lock()
	defer r.subMu.Unlock()
	if r.subActive {
		return
	}
	r.stop = make(chan struct{})
	r.subActive = true
	go r.subscribe()
}
func (r *Room) subscribe() {
	defer func() {
		r.subMu.Lock()
		r.subActive = false
		r.subMu.Unlock()
	}()

	ch := "room:" + r.Code + ":pub"
	ps := rdb.Subscribe(context.Background(), ch)
	r.pubsub = ps
	defer ps.Close()

	for {
		select {
		case <-r.stop:
			return
		case m, ok := <-ps.Channel():
			if !ok {
				return
			}
			r.broadcast <- []byte(m.Payload)
		}
	}
}
func (r *Room) stopSubscriber() {
	r.subMu.Lock()
	defer r.subMu.Unlock()
	if !r.subActive {
		return
	}
	close(r.stop)
	if r.pubsub != nil {
		_ = r.pubsub.Close()
		r.pubsub = nil
	}
	r.subActive = false
}

func (r *Room) run() {
	for {
		select {
		case c := <-r.register:

			r.ensureSubscriber()

			r.mu.Lock()
			r.Clients[c] = struct{}{}
			r.mu.Unlock()
			//_ = c.conn.WriteJSON(map[string]any{
			//	"type": "hello",
			//	"data": map[string]string{"room": r.Code, "name": c.name},
			//})
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
				if c.Ready && r.Ready > 0 {
					r.Ready--
				}
				close(c.send)
			}
			empty := len(r.Clients) == 0

			r.mu.Unlock()
			if empty {
				r.stopSubscriber()
			}

		case msg := <-r.broadcast:
			r.mu.RLock()
			n := len(r.Clients)
			for cl := range r.Clients {
				select {
				case cl.send <- msg:
				default:
				}
			}
			r.mu.RUnlock()
			log.Printf("[room %s] fan-out to %d clients", r.Code, n)

		}
	}
}
