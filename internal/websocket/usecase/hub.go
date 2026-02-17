package usecase

import (
	"sync"

	"notification-srv/pkg/log"
)

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	// Registered clients.
	clients map[*Connection]bool

	// User to connections mapping for targeted messaging.
	// user_id -> set of connections
	users map[string]map[*Connection]bool

	// Inbound messages from the connections.
	broadcast chan []byte

	// Register requests from the connections.
	register chan *Connection

	// Unregister requests from connections.
	unregister chan *Connection

	// Lock for maps
	mu sync.RWMutex

	logger log.Logger
}

func newHub(logger log.Logger, maxConnections int) *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Connection),
		unregister: make(chan *Connection),
		clients:    make(map[*Connection]bool),
		users:      make(map[string]map[*Connection]bool),
		logger:     logger,
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if _, ok := h.users[client.userID]; !ok {
				h.users[client.userID] = make(map[*Connection]bool)
			}
			h.users[client.userID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				if userConns, ok := h.users[client.userID]; ok {
					delete(userConns, client)
					if len(userConns) == 0 {
						delete(h.users, client.userID)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToUser sends a message to all active connections of a specific user.
func (h *Hub) SendToUser(userID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if conns, ok := h.users[userID]; ok {
		for client := range conns {
			select {
			case client.send <- message:
			default:
				// Buffer full or connection dead, we might close it here or let the writePump handle it
				// For safety in this tight loop, we skip blocking
			}
		}
	}
}

// Broadcast sends a message to all active connections.
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// Stats returns the current statistics of the hub.
func (h *Hub) Stats() (int, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients), len(h.users)
}
