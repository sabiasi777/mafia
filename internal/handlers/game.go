package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sabiasi777/mafia/internal/logic"
	"github.com/sabiasi777/mafia/internal/models"
)

func (rm *RoomManager) StartGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.StartRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	roomCode := req.RoomCode
	username := req.CurrentUser

	if username != rm.Rooms[roomCode].Owner {
		http.Error(w, "Only room owner can start", http.StatusForbidden)
		return
	}

	room, exists := rm.Rooms[roomCode]
	if !exists {
		http.NotFound(w, r)
		return
	}

	rm.Rooms[roomCode].CurrentUser = req.CurrentUser
	room.CurrentSpeakerIndex = 0
	fmt.Println("rm.Rooms[roomCode].CurrentUser", rm.Rooms[roomCode].CurrentUser)

	logic.AssignRoles(room)

	rm.BroadcastGameStart(roomCode)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Game starting..."))
}
