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
	room, exists := rm.Rooms[roomCode]
	if !exists {
		http.NotFound(w, r)
		return
	}

	rm.Rooms[roomCode].CurrentUser = req.CurrentUser

	logic.AssignRoles(room)
	fmt.Println("LETS SAY THE ROLE:")
	fmt.Println(room.Players[len(room.Players)-1].Role, room.Players[len(room.Players)-1].Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Players)
}
