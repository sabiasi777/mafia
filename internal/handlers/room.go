package handlers

import (
	"fmt"
	"net/http"

	"github.com/sabiasi777/mafia/internal/logic"
	"github.com/sabiasi777/mafia/internal/models"
)

func (rm *RoomManager) JoinHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Joinhandler")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("join")

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	roomcode := r.FormValue("roomcode")

	fmt.Println("RoomCode:", roomcode)
	fmt.Println("Username:", username)

	room, exists := rm.Rooms[roomcode]
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	room.Players = append(room.Players, models.Player{Name: username, IsActive: true})

	http.Redirect(w, r, "/room/"+roomcode+"?user="+username, http.StatusSeeOther)
}

func (rm *RoomManager) CreateRoom(w http.ResponseWriter, r *http.Request) {
	fmt.Println("CreateRoom")
	fmt.Println(r.Method)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	roomCode := logic.GenerateRoomCode(6)

	room := models.Room{
		Code:    roomCode,
		Owner:   username,
		Players: []models.Player{},
	}

	room.Players = append(room.Players, models.Player{Name: username, IsActive: true})
	rm.Rooms[roomCode] = &room
	fmt.Println("Room Created:", rm.Rooms[roomCode])

	http.Redirect(w, r, "/room/"+roomCode+"?user="+username, http.StatusSeeOther)
}

func (rm *RoomManager) RoomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	roomCode := r.URL.Path[len("/room/"):]
	fmt.Println("ROOMCODE IN ROOM HANDLER:", roomCode)
	room, exists := rm.Rooms[roomCode]
	if !exists {
		http.NotFound(w, r)
		return
	}

	room.ActiveRoles = logic.GetActiveRoles(len(room.Players))
	fmt.Println("Room currentUser:", room.CurrentUser)
	fmt.Printf("Room Owner: '%s'\n", room.Owner)
	fmt.Println("ActiveRoles", room.ActiveRoles)

	if err := rm.Tmpl.ExecuteTemplate(w, "game.html", room); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}

func (rm *RoomManager) getCurrentPlayers(roomCode string) []models.Player {
	return rm.Rooms[roomCode].Players
}
