package main

import (
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	RoleCrew Role = "crew"
	RoleMod  Role = "mod"
)

var (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Role string
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

func NewClient(conn *websocket.Conn, name string, room *Room) *Client {
	n := strings.TrimSpace(name)
	if n == "" {
		return nil
	}
	return &Client{
		ID:    uuid.New().String(),
		conn:  conn,
		name:  n,
		Room:  room,
		send:  make(chan []byte, 256),
		Alive: true,
		Ready: true,
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
		_ = c.conn.WriteMessage(websocket.TextMessage, message)
		log.Printf("Message from Client: %s\n", message)
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
		case message, _ := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(pongWait))
			log.Printf("sending message to client: %s\n", message)
			_ = c.conn.WriteMessage(websocket.TextMessage, message)
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(pongWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
