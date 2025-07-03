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

func (rm *RoomManager) HandleChat(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("room")
	userName := r.URL.Query().Get("user")

	fmt.Println("RoomCode in handleChat", roomCode)
	fmt.Println("userName in handleChat", userName)

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

	fmt.Println("RoomsConnections[roomCode]", roomsConnections[roomCode])

	for name, clientConn := range roomsConnections[roomCode] {
		if name != userName {
			joinMsg := models.SignalingMessage{Type: "player-joined", Name: userName}
			payload, _ := json.Marshal(joinMsg)
			clientConn.WriteMessage(websocket.TextMessage, payload)

			playerListMsg := models.SignalingMessage{Type: "player-list-update", Players: rm.getCurrentPlayers(roomCode)}
			listPayLoad, _ := json.Marshal(playerListMsg)
			clientConn.WriteMessage(websocket.TextMessage, listPayLoad)

			// Remove this list variables and update case: "player-join" for both join and update
		}
	}

	roomsConnections[roomCode][userName] = conn
	fmt.Printf("User '%s' joined room '%s'\n", userName, roomCode)
	fmt.Println("RoomsConnections[roomCode]", roomsConnections[roomCode])
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
	fmt.Println("HandleConnection")
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("Read error for user '%s': %v\n", senderName, err)
			}
			break
		}

		fmt.Println("conn.ReadMessage():", string(msg))

		var message models.SignalingMessage
		if err := json.Unmarshal(msg, &message); err != nil {
			fmt.Println("Error unmarshaling message:", err)
			continue
		}

		message.Sender = senderName
		fmt.Println("message.Receiver:", message.Receiver)

		mu.Lock()
		room, ok := roomsConnections[roomCode]
		if !ok {
			mu.Unlock()
			continue
		}

		fmt.Println("roomsConnections[roomCode]:", room)

		if message.Receiver != "" {
			if targetConn, ok := room[message.Receiver]; ok {
				fmt.Printf("Relaying message from %s to %s\n", senderName, message.Receiver)
				if err := targetConn.WriteMessage(websocket.TextMessage, msg); err != nil {
					fmt.Printf("Error sending private message to %s: %v\n", message.Receiver, err)
				}
			} else {
				fmt.Printf("Receiver %s not found in room %s\n", message.Receiver, roomCode)
			}
		} else {
			fmt.Printf("Broadcasting message from %s\n", senderName)
			for name, clientConn := range room {
				if err := clientConn.WriteMessage(websocket.TextMessage, msg); err != nil {
					fmt.Printf("Error broadcasting to user %s: %v\n", name, err)
				}
			}
		}
		mu.Unlock()
	}
}
