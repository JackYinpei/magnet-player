package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a connected client (producer or consumer)
type Client struct {
	ID   string
	Conn *websocket.Conn
	Type string // "producer" or "consumer"
}

// Message represents the structure of messages exchanged with clients
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

var (
	clients    = make(map[string]*Client)
	clientsMux sync.Mutex
	upgrader   = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all connections for simplicity
		},
	}
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Register client with a unique ID
	clientID := r.URL.Query().Get("id")
	clientType := r.URL.Query().Get("type")
	
	if clientID == "" || (clientType != "producer" && clientType != "consumer") {
		log.Printf("Invalid client parameters")
		return
	}

	// Register client
	clientsMux.Lock()
	client := &Client{
		ID:   clientID,
		Conn: conn,
		Type: clientType,
	}
	clients[clientID] = client
	clientsMux.Unlock()

	log.Printf("Client connected: %s (%s)", clientID, clientType)

	// Handle client messages
	for {
		// Read message from the client
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		// Handle message based on type
		switch msg.Type {
		case "offer", "answer", "ice-candidate":
			// Forward message to the other client
			forwardMessage(clientID, msgBytes)
		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}

	// Unregister client when disconnected
	clientsMux.Lock()
	delete(clients, clientID)
	clientsMux.Unlock()
	log.Printf("Client disconnected: %s (%s)", clientID, clientType)
}

func forwardMessage(senderID string, msg []byte) {
	clientsMux.Lock()
	defer clientsMux.Unlock()

	sender := clients[senderID]
	if sender == nil {
		return
	}

	// Determine target type (send producer messages to consumer and vice versa)
	targetType := "consumer"
	if sender.Type == "consumer" {
		targetType = "producer"
	}

	// Forward message to all clients of the target type
	for _, client := range clients {
		if client.Type == targetType {
			if err := client.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("Error forwarding message to %s: %v", client.ID, err)
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	log.Printf("Starting signaling server on :8090")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
