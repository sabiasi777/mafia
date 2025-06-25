package handlers

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var roomsConnections = make(map[string]map[*websocket.Conn]bool)
var mu sync.Mutex

func HandleChat(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("room")
	if roomCode == "" {
		http.Error(w, "Missing room code", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Websocket upgrade error:", err)
		return
	}

	mu.Lock()
	if roomsConnections[roomCode] == nil {
		roomsConnections[roomCode] = make(map[*websocket.Conn]bool)
	}
	roomsConnections[roomCode][conn] = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		if clients, exists := roomsConnections[roomCode]; exists {
			delete(clients, conn)
			if len(clients) == 0 {
				delete(roomsConnections, roomCode)
			}
		}
		mu.Unlock()
		conn.Close()
	}()

	handleConnection(conn, roomCode)
}

func handleConnection(conn *websocket.Conn, roomCode string) {
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Read error:", err)
			break
		}

		mu.Lock()
		for client := range roomsConnections[roomCode] {
			if msgType == websocket.TextMessage {
				if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
					fmt.Println("Error writing message:", err)
				}
			}
		}
		mu.Unlock()
	}
}
