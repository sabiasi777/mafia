package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sabiasi777/mafia/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var roomsConnections = make(map[string]map[string]*websocket.Conn)
var mu sync.Mutex

func HandleChat(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("room")
	userName := r.URL.Query().Get("user")

	if roomCode == "" || userName == "" {
		http.Error(w, "Missing room code or user name", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Websocket upgrade error:", err)
		return
	}

	mu.Lock()
	if roomsConnections[roomCode] == nil {
		roomsConnections[roomCode] = make(map[string]*websocket.Conn)
	}

	for name, clientConn := range roomsConnections[roomCode] {
		if name != userName {
			joinMsg := models.SignalingMessage{Type: "player-joined", Name: userName}
			payload, _ := json.Marshal(joinMsg)
			clientConn.WriteMessage(websocket.TextMessage, payload)
		}
	}

	roomsConnections[roomCode][userName] = conn
	fmt.Printf("User '%s' joined room '%s'\n", userName, roomCode)
	mu.Unlock()

	defer func() {
		mu.Lock()
		if room, exists := roomsConnections[roomCode]; exists {
			delete(room, userName)
			if len(room) == 0 {
				delete(roomsConnections, roomCode)
				fmt.Printf("Room '%s' is now empty and closed.\n", roomCode)
			}
		}
		mu.Unlock()
		conn.Close()
		fmt.Printf("Connection for user '%s' closed.\n", userName)
	}()

	handleConnection(conn, roomCode, userName)
}

func handleConnection(conn *websocket.Conn, roomCode string, senderName string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("Read error for user '%s': %v\n", senderName, err)
			}
			break
		}

		var message models.SignalingMessage
		if err := json.Unmarshal(msg, &message); err != nil {
			fmt.Println("Error unmarshaling message:", err)
			continue
		}

		message.Sender = senderName

		mu.Lock()
		room, ok := roomsConnections[roomCode]
		if !ok {
			mu.Unlock()
			continue
		}

		if message.Receiver != "" {
			if targetConn, ok := room[message.Receiver]; ok {
				if err := targetConn.WriteMessage(websocket.TextMessage, msg); err != nil {
					fmt.Printf("Error sending private message to %s: %v\n", message.Receiver, err)
				}
			} else {
				fmt.Printf("Receiver %s not found in room %s\n", message.Receiver, roomCode)
			}
		} else {
			for name, clientConn := range room {
				if name != senderName && message.Type == "text" {
					if err := clientConn.WriteMessage(websocket.TextMessage, msg); err != nil {
						fmt.Println("Error broadcasting message:", err)
					}
				}
			}
		}
		mu.Unlock()
	}
}
