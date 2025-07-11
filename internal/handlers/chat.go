package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

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

func (rm *RoomManager) HandleChat(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("room")
	userName := r.URL.Query().Get("user")

	fmt.Println("Handle Chat")

	if roomCode == "" || userName == "" {
		http.Error(w, "Missing room code or user name", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Websocket upgrade error:", err)
		return
	}

	rm.mu.Lock()
	if rm.Connections[roomCode] == nil {
		rm.Connections[roomCode] = make(map[string]*websocket.Conn)
	}

	rm.Connections[roomCode][userName] = conn

	for name, clientConn := range rm.Connections[roomCode] {
		if name != userName {
			playerListMsg := models.SignalingMessage{
				Type:    "player-list-update",
				Players: rm.getCurrentPlayers(roomCode),
				Name:    userName,
			}
			listPayLoad, _ := json.Marshal(playerListMsg)
			clientConn.WriteMessage(websocket.TextMessage, listPayLoad)
		}
	}
	rm.mu.Unlock()

	defer func() {
		rm.mu.Lock()

		if room, exists := rm.Connections[roomCode]; exists {
			delete(room, userName)
			players := rm.Rooms[roomCode].Players

			for i, player := range players {
				if player.Name == userName {
					rm.Rooms[roomCode].Players = append(players[:i], players[i+1:]...)
					break
				}
			}
			if len(room) == 0 {
				delete(rm.Connections, roomCode)
				fmt.Printf("Room '%s' is now empty and closed.\n", roomCode)
			}

			fmt.Printf("%s left the room: %v\n", userName, room)
			fmt.Println("getting current players", rm.getCurrentPlayers(roomCode))

			leaveMsg := models.SignalingMessage{
				Type:    "player-list-update",
				Name:    userName,
				Players: rm.getCurrentPlayers(roomCode),
			}
			payload, _ := json.Marshal(leaveMsg)

			for _, clientConn := range room {
				clientConn.WriteMessage(websocket.TextMessage, payload)
			}
		}
		rm.mu.Unlock()
		conn.Close()
		fmt.Printf("Connection for user '%s' closed.\n", userName)
	}()

	rm.handleConnection(conn, roomCode, userName)
}

func (rm *RoomManager) handleConnection(conn *websocket.Conn, roomCode string, senderName string) {
	fmt.Println("HandleConnection")
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
		fmt.Println("message.Receiver:", message.Receiver)
		fmt.Println("message.Sender:", message.Sender)
		fmt.Println("message:", message)

		if message.Type == "finish-speech" {
			room, ok := rm.Rooms[roomCode]
			if !ok {
				rm.mu.Unlock()
				continue
			}

			if senderName == room.Players[room.CurrentSpeakerIndex].Name {
				room.CurrentSpeakerIndex++

				if room.CurrentSpeakerIndex >= len(room.Players) {
					room.CurrentSpeakerIndex = 0
				}

				rm.mu.Unlock()
				go rm.BroadcastTurnUpdate(roomCode)
			} else {
				rm.mu.Unlock()
			}

			continue
		}

		rm.mu.Lock()
		connections, ok := rm.Connections[roomCode]
		if !ok {
			rm.mu.Unlock()
			continue
		}

		if message.Receiver != "" {
			if targetConn, ok := connections[message.Receiver]; ok {
				fmt.Printf("Relaying message from %s to %s\n", senderName, message.Receiver)
				if err := targetConn.WriteMessage(websocket.TextMessage, msg); err != nil {
					fmt.Printf("Error sending private message to %s: %v\n", message.Receiver, err)
				}
			} else {
				fmt.Printf("Receiver %s not found in room %s\n", message.Receiver, roomCode)
			}
		} else {
			fmt.Printf("Broadcasting message from %s\n", senderName)
			for name, clientConn := range connections {
				if message.Type == "text" || name != senderName {
					if err := clientConn.WriteMessage(websocket.TextMessage, msg); err != nil {
						fmt.Printf("Error broadcasting to user %s: %v\n", name, err)
					}
				}
			}
		}
		rm.mu.Unlock()
	}
}

func (rm *RoomManager) BroadcastGameStart(roomCode string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, roomExists := rm.Rooms[roomCode]
	connections, connectionsExist := rm.Connections[roomCode]

	if !roomExists || !connectionsExist {
		fmt.Println("Broadcast Error: Room or connections not found for", roomCode)
		return
	}

	for _, player := range room.Players {
		if conn, ok := connections[player.Name]; ok {
			fmt.Printf("player from room.Players: %v\n", player)
			message := models.SignalingMessage{
				Type: "game-start",
				Me: &models.Player{
					Name: player.Name,
					Role: player.Role,
				},
				Players: rm.getCurrentPlayers(roomCode),
			}

			payload, err := json.Marshal(message)
			if err != nil {
				fmt.Printf("Error marshaling game-start message for %s: %v\n", player.Name, err)
				continue
			}

			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				fmt.Printf("Error sending game-start message to %s: %v\n", player.Name, err)
			} else {
				fmt.Printf("Sent game-start message to %s with role %s\n", player.Name, player.Role)
			}
		}
	}
}

func (rm *RoomManager) BroadcastTurnUpdate(roomCode string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, roomExists := rm.Rooms[roomCode]
	connections, connectionsExists := rm.Connections[roomCode]

	if !roomExists || !connectionsExists {
		return
	}

	currentSpeaker := room.Players[room.CurrentSpeakerIndex]

	message := models.SignalingMessage{
		Type:        "turn-update",
		SpeakerName: currentSpeaker.Name,
	}
	payload, _ := json.Marshal(message)

	for _, conn := range connections {
		conn.WriteMessage(websocket.TextMessage, payload)
	}

}
