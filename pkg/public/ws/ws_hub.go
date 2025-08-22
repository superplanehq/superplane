package ws

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
    writeWait = 10 * time.Second       // Time allowed to write a message
    pongWait  = 20 * time.Second       // Must receive pong within 20s
    pingPeriod = 10 * time.Second      // Ping every 10s
)

// Client represents a connected websocket client
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	Done     chan struct{}
	canvasID string // Which canvas this client is watching
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Map of canvas IDs to clients subscribed to that canvas
	canvasSubscriptions map[string]map[*Client]bool

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Guards access to maps
	mutex sync.RWMutex
}

// NewHub creates a new hub
func NewHub() *Hub {
	return &Hub{
		clients:             make(map[*Client]bool),
		canvasSubscriptions: make(map[string]map[*Client]bool),
		register:            make(chan *Client),
		unregister:          make(chan *Client),
		mutex:               sync.RWMutex{},
	}
}

// Run starts the hub processing loop
func (h *Hub) Run() {
	go func() {
		for {
			select {
			case client := <-h.register:
				h.registerClient(client)
			case client := <-h.unregister:
				h.unregisterClient(client)
			}
		}
	}()
}

// registerClient adds a new client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client] = true

	if _, ok := h.canvasSubscriptions[client.canvasID]; !ok {
		h.canvasSubscriptions[client.canvasID] = make(map[*Client]bool)
	}
	h.canvasSubscriptions[client.canvasID][client] = true
	log.Debugf("Client subscribed to canvas: %s", client.canvasID)

	log.Debugf("New client registered %v, total clients: %d", client, len(h.clients))
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// If this client has a connection, close it
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Also remove from canvas subscriptions
		if client.canvasID != "" {
			if clients, ok := h.canvasSubscriptions[client.canvasID]; ok {
				delete(clients, client)

				// If this was the last client for this canvas, remove the canvas entry
				if len(clients) == 0 {
					delete(h.canvasSubscriptions, client.canvasID)
				}
			}
		}
		log.Debugf("Client unregistered, remaining clients: %d", len(h.clients))
	}
}

// BroadcastAll sends a message to all connected clients
func (h *Hub) BroadcastAll(message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			// If the client's buffer is full, assume it's gone and unregister it
			h.unregisterClient(client)
		}
	}
}

// BroadcastToCanvas sends a message to all clients subscribed to a specific canvas
func (h *Hub) BroadcastToCanvas(canvasID string, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Get clients subscribed to this canvas
	if clients, ok := h.canvasSubscriptions[canvasID]; ok {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				// If the client's buffer is full, assume it's gone and unregister it
				h.unregisterClient(client)
			}
		}
	}
}

// NewClient creates a new websocket client
func (h *Hub) NewClient(conn *websocket.Conn, canvasID string) *Client {
	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 4096),
		Done:     make(chan struct{}),
		canvasID: canvasID,
	}

	// Register this client with the hub
	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	return client
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
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

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		close(c.Done)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(1024 * 1024) // 1MB max message size
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Process incoming messages
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("Unexpected WebSocket closure: %v", err)
			} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				// Normal closure cases
				log.Info("WebSocket closed normally")
			} else if err == io.EOF {
				// EOF is common when the client disconnects without sending a close frame
				log.Info("WebSocket connection EOF")
			} else if ne, ok := err.(*net.OpError); ok {
				// Handle network operation errors
				log.Warnf("WebSocket network error: %v", ne)
			} else {
				// Catch any other errors
				log.Warnf("WebSocket read error: %v", err)
			}
			break
		}

		// Handle client messages
		c.handleMessage(message)
	}
}

// handleMessage processes incoming messages from clients
func (c *Client) handleMessage(message []byte) {
	// Handle client messages
	log.Infof("Received message: %s", string(message))
	return
}
