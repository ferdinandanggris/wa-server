package webhook

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketMessage is the JSON payload sent through WebSocket connections.
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Client represents a single WebSocket connection belonging to a company.
type Client struct {
	ID        string
	CompanyID string
	Conn      *websocket.Conn
	Send      chan []byte
	closeOnce sync.Once
}

// WebSocketHub manages WebSocket connections grouped by company.
type WebSocketHub struct {
	clients    map[string]*Client
	companyMap map[string]map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *CompanyBroadcast
	mu         sync.RWMutex
}

// CompanyBroadcast targets a message to all clients of a specific company.
type CompanyBroadcast struct {
	CompanyID string
	Message   WebSocketMessage
}

// NewWebSocketHub creates and initializes a new WebSocketHub.
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[string]*Client),
		companyMap: make(map[string]map[string]*Client),
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		broadcast:  make(chan *CompanyBroadcast, 100),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if _, ok := h.companyMap[client.CompanyID]; !ok {
				h.companyMap[client.CompanyID] = make(map[string]*Client)
			}
			h.companyMap[client.CompanyID][client.ID] = client
			h.mu.Unlock()
			slog.Info("client registered", "client_id", client.ID, "company_id", client.CompanyID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				close(client.Send)
				delete(h.clients, client.ID)
				delete(h.companyMap[client.CompanyID], client.ID)
			}
			h.mu.Unlock()
			slog.Info("client unregistered", "client_id", client.ID)

		case broadcast := <-h.broadcast:
			h.mu.RLock()
			clients, ok := h.companyMap[broadcast.CompanyID]
			h.mu.RUnlock()
			if !ok {
				continue
			}

			data, err := json.Marshal(broadcast.Message)
			if err != nil {
				slog.Error("failed to marshal message", "error", err)
				continue
			}

			for _, client := range clients {
				select {
				case client.Send <- data:
				default:
					h.mu.Lock()
					delete(h.clients, client.ID)
					delete(h.companyMap[client.CompanyID], client.ID)
					h.mu.Unlock()
					close(client.Send)
				}
			}
		}
	}
}

// BroadcastToCompany sends a message to all WebSocket clients of a company.
func (h *WebSocketHub) BroadcastToCompany(companyID string, msg WebSocketMessage) {
	h.broadcast <- &CompanyBroadcast{
		CompanyID: companyID,
		Message:   msg,
	}
}

// HandleWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *WebSocketHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	companyID := r.URL.Query().Get("company_id")
	if companyID == "" {
		http.Error(w, "company_id required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("failed to upgrade websocket", "error", err)
		return
	}

	client := &Client{
		ID:        generateClientID(),
		CompanyID: companyID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
	}

	h.register <- client

	go client.writePump()
	go client.readPump(h)
}

func (h *WebSocketHub) unregisterClient(client *Client) {
	h.unregister <- client
}

func (c *Client) closeConn() {
	c.closeOnce.Do(func() {
		c.Conn.Close()
	})
}

func (c *Client) readPump(hub *WebSocketHub) {
	defer func() {
		hub.unregisterClient(c)
		c.closeConn()
	}()

	c.Conn.SetReadLimit(512)
	if err := c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		slog.Error("failed to set read deadline", "error", err)
	}
	c.Conn.SetPongHandler(func(string) error {
		if err := c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			slog.Error("failed to set read deadline in pong handler", "error", err)
		}
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("websocket error", "error", err, "client_id", c.ID)
			}
			break
		}

		slog.Debug("received message from client", "client_id", c.ID, "message", string(message))
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.closeConn()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				slog.Error("failed to set write deadline", "error", err)
				return
			}
			if !ok {
				if err := c.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					slog.Error("failed to write close message", "error", err)
				}
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				slog.Error("failed to write message", "error", err)
				return
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				slog.Error("failed to set write deadline for ping", "error", err)
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
