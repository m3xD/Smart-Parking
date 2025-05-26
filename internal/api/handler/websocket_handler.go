package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"smart_parking/internal/domain"
	"sync"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Cho phép kết nối từ mọi nguồn
	},
}

type WebSocketManager struct {
	clients    map[*websocket.Conn]bool // Kết nối WebSocket hiện tại
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan []byte
	mutex      sync.RWMutex
}

func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan []byte),
	}
}

func (wsm *WebSocketManager) Start() {
	for {
		select {
		case client := <-wsm.register:
			wsm.mutex.Lock()
			wsm.clients[client] = true
			wsm.mutex.Unlock()
			log.Printf("WebSocket client connected. Total: %d", len(wsm.clients))

		case client := <-wsm.unregister:
			wsm.mutex.Lock()
			if _, ok := wsm.clients[client]; ok {
				delete(wsm.clients, client)
				client.Close()
			}
			wsm.mutex.Unlock()
			log.Printf("WebSocket client disconnected. Total: %d", len(wsm.clients))

		case message := <-wsm.broadcast:
			wsm.mutex.RLock()
			for client := range wsm.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error writing to WebSocket client: %v", err)
					client.Close()
					delete(wsm.clients, client)
				}
			}
			wsm.mutex.RUnlock()
		}
	}
}

func (wsm *WebSocketManager) BroadcastGateEvent(event domain.GateEventNotification) {
	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling gate event: %v", err)
		return
	}

	select {
	case wsm.broadcast <- message:
	default:
		log.Println("Broadcast channel is full, dropping message")
	}
}

type WebSocketHandler struct {
	wsManager *WebSocketManager
}

func NewWebSocketHandler(wsManager *WebSocketManager) *WebSocketHandler {
	return &WebSocketHandler{wsManager: wsManager}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	h.wsManager.register <- conn

	// Keep connection alive và handle disconnect
	go func() {
		defer func() {
			h.wsManager.unregister <- conn
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}
		}
	}()
}
