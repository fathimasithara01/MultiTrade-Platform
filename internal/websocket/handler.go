package websocket

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development and local testing
		return true
	},
}

// Handler handles WebSocket upgrade requests.
type Handler struct {
	hub *Hub
}

// NewHandler constructs a Handler.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// Connect upgrades HTTP connection and registers WebSocket client.
func (h *Handler) Connect(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade HTTP connection to WebSocket")
		return
	}

	client := NewClient(h.hub, conn)
	h.hub.RegisterChan <- client

	// Start read/write pumps in separate goroutines
	go client.WritePump()
	go client.ReadPump()

	log.Info().Msg("New WebSocket client connected successfully")
}
