package websocket

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	// Active clients mapped to their subscription topics.
	// Map key is the topic name (e.g., "asset:123"), value is a set of Clients.
	subscriptions map[string]map[*Client]bool
	subMutex      sync.RWMutex

	// Inbound messages from clients or Redis.
	BroadcastChan chan BroadcastMessage

	// Register requests from clients.
	RegisterChan chan *Client

	// Unregister requests from clients.
	UnregisterChan chan *Client
}

// BroadcastMessage holds the target topic and the message content.
type BroadcastMessage struct {
	Topic   string
	Message []byte
}

// NewHub constructs a new Hub.
func NewHub() *Hub {
	return &Hub{
		subscriptions:  make(map[string]map[*Client]bool),
		BroadcastChan:  make(chan BroadcastMessage, 4096),
		RegisterChan:   make(chan *Client, 256),
		UnregisterChan: make(chan *Client, 256),
	}
}

// Run starts the event loop for register, unregister, and broadcasting.
func (h *Hub) Run() {
	log.Info().Msg("WebSocket Hub started")
	for {
		select {
		case <-h.RegisterChan:
			log.Debug().Msg("WebSocket client registered with hub")
			// No logic needed here, client handles its own connection lifecyle

		case client := <-h.UnregisterChan:
			h.unsubscribeAll(client)
			close(client.Send)
			log.Debug().Msg("WebSocket client unregistered from hub")

		case bm := <-h.BroadcastChan:
			h.subMutex.RLock()
			clients, exists := h.subscriptions[bm.Topic]
			if exists {
				for client := range clients {
					select {
					case client.Send <- bm.Message:
					default:
						// If client buffer is full, unregister it
						log.Warn().Msg("Client write buffer full, unregistering client")
						h.unsubscribeAll(client)
						close(client.Send)
					}
				}
			}
			h.subMutex.RUnlock()
		}
	}
}

// Subscribe adds a client to a specific topic subscriptions.
func (h *Hub) Subscribe(client *Client, topic string) {
	h.subMutex.Lock()
	defer h.subMutex.Unlock()

	if h.subscriptions[topic] == nil {
		h.subscriptions[topic] = make(map[*Client]bool)
	}
	h.subscriptions[topic][client] = true
	client.topics[topic] = true
	log.Info().Str("topic", topic).Msg("Client subscribed to topic")
}

// Unsubscribe removes a client from a specific topic subscriptions.
func (h *Hub) Unsubscribe(client *Client, topic string) {
	h.subMutex.Lock()
	defer h.subMutex.Unlock()

	if clients, exists := h.subscriptions[topic]; exists {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.subscriptions, topic)
		}
	}
	delete(client.topics, topic)
	log.Info().Str("topic", topic).Msg("Client unsubscribed from topic")
}

// unsubscribeAll cleans up all subscriptions for a client.
func (h *Hub) unsubscribeAll(client *Client) {
	h.subMutex.Lock()
	defer h.subMutex.Unlock()

	for topic := range client.topics {
		if clients, exists := h.subscriptions[topic]; exists {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.subscriptions, topic)
			}
		}
	}
	client.topics = make(map[string]bool)
}
