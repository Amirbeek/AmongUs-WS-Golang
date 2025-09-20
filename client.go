package main

import (
	"context"
	"encoding/json"
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
		key := "room:" + c.Room.Code
		ctx := context.Background()

		if err := rdb.RPush(ctx, key, string(message)).Err(); err != nil {
			log.Println("Redis RPush error:", err)
		}
		if err := rdb.LTrim(ctx, key, -50, -1).Err(); err != nil {
			log.Println("Redis LTrim error:", err)
		}

		if c.Room.active >= 3 {
			var data map[string]interface{}
			if err := json.Unmarshal(message, &data); err != nil {
				log.Println("decode error:", err)
				continue
			}

			if t, ok := data["type"].(string); ok && t == "agree" && !c.Ready {
				c.Ready = true
				c.Room.mu.Lock()
				c.Room.Ready++
				c.Room.mu.Unlock()

				log.Printf("%s is ready", c.name)
				msg, _ := json.Marshal(map[string]interface{}{
					"type": "agree",
					"data": map[string]string{
						"username": c.name,
					},
				})
				c.Room.broadcast <- msg

				if c.Room.allReady() {
					go c.Room.startGameCountDown()
				}
				continue
			}

		}
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
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
