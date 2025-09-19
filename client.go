package main

import (
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Role string

const (
	RoleCrew   Role = "crew"
	RoleKiller Role = "killer"
)

type Client struct {
	ID    string
	name  string
	conn  *websocket.Conn
	Room  *Room
	send  chan []byte
	Alive bool
	Ready bool
	Role  Role
}

var (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

func NewClient(conn *websocket.Conn, room *Room, name string) *Client {
	n := strings.TrimSpace(name)
	if n == "" {
		n = "Player"
	}
	return &Client{
		ID:    uuid.New().String(),
		name:  n,
		conn:  conn,
		Room:  room,
		send:  make(chan []byte, 256),
		Alive: true,
		Ready: false,
		Role:  RoleCrew,
	}
}

func (c *Client) readPump() {
	defer func() {
		c.Room.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(4 << 10)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

		log.Printf("recv: %s", message)
		c.Room.broadcast <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			log.Printf("send: %s", msg)
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			// Control frame sifatida ping
			// _ = c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
