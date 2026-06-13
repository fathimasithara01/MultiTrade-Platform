package notification

import (
	"context"
	"encoding/json"

	"github.com/fathimasithara01/multitrade-platform/internal/websocket"
)

// WebsocketNotifier broadcasts structured messages to connected WebSocket clients.
type WebsocketNotifier struct {
	hub *websocket.Hub
}

// NewWebsocketNotifier constructs a WebsocketNotifier.
func NewWebsocketNotifier(hub *websocket.Hub) *WebsocketNotifier {
	return &WebsocketNotifier{hub: hub}
}

// Broadcast serialises payload and sends it to all subscribers on topic.
func (n *WebsocketNotifier) Broadcast(_ context.Context, topic string, payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	n.hub.BroadcastChan <- websocket.BroadcastMessage{
		Topic:   topic,
		Message: b,
	}
	return nil
}
