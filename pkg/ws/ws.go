// Package ws provides WebSocket support for Kashvi using gorilla/websocket.
//
// # Quick start
//
//	// In your route file:
//	router.Get("/ws/chat", "ws.chat", ctx.Wrap(func(c *ctx.Context) {
//	    ws.Upgrade(c.W, c.R, ChatHub)
//	}))
//
//	// Define a hub and start it:
//	var ChatHub = ws.NewHub()
//	func init() { go ChatHub.Run() }
//
//	// Broadcast from anywhere:
//	ChatHub.Broadcast <- []byte("hello everyone")
package ws

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shashiranjanraj/kashvi/pkg/logger"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512 KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins by default — restrict in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// SetCheckOrigin replaces the default (allow-all) origin checker.
func SetCheckOrigin(fn func(r *http.Request) bool) {
	upgrader.CheckOrigin = fn
}

// ─── Client ───────────────────────────────────────────────────────────────────

// Client represents a single connected WebSocket client.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// readPump pumps messages from the WebSocket connection to the hub.
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
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warn("ws: unexpected close", "error", err)
			}
			break
		}
		c.hub.Inbound <- Message{Client: c, Data: msg}
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
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

// Send queues a message to be sent to this specific client.
func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
	default:
		// Buffer full — drop message.
	}
}

// ─── Hub ──────────────────────────────────────────────────────────────────────

// Message is an inbound message received from a client.
type Message struct {
	Client *Client
	Data   []byte
}

// Hub maintains all active WebSocket connections and handles broadcasting.
type Hub struct {
	clients    map[*Client]bool
	Broadcast  chan []byte  // send to all connected clients
	Inbound    chan Message // messages received from clients
	register   chan *Client
	unregister chan *Client
	// OnMessage is called for every inbound message (optional).
	OnMessage func(hub *Hub, msg Message)
}

// NewHub creates a new Hub. Call hub.Run() in a goroutine at startup.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte, 256),
		Inbound:    make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub event loop. Must be run in its own goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			logger.Info("ws: client connected", "total", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				logger.Info("ws: client disconnected", "total", len(h.clients))
			}

		case msg := <-h.Broadcast:
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}

		case msg := <-h.Inbound:
			if h.OnMessage != nil {
				h.OnMessage(h, msg)
			}
		}
	}
}

// ClientCount returns the number of currently connected clients.
func (h *Hub) ClientCount() int { return len(h.clients) }

// ─── Upgrade ─────────────────────────────────────────────────────────────────

// Upgrade upgrades an HTTP connection to a WebSocket and registers the
// resulting client with the given hub.
func Upgrade(w http.ResponseWriter, r *http.Request, hub *Hub) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("ws: upgrade failed", "error", err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	hub.register <- client
	go client.writePump()
	go client.readPump()
}
