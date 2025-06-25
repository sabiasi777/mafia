package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sabiasi777/mafia/internal/handlers"
	"github.com/sabiasi777/mafia/internal/models"
)

func main() {
	manager := handlers.RoomManager{
		Rooms: make(map[string]*models.Room),
		Tmpl:  template.Must(template.ParseGlob("templates/*.html")),
	}

	permanentRoomCode := "ADMIN"
	room := models.Room{
		Code: permanentRoomCode,
		Players: []models.Player{
			{Name: "Saba", Role: "Villager", IsActive: true},
			{Name: "Beqa", Role: "Villager", IsActive: true},
			{Name: "Nodo", Role: "Doctor", IsActive: true},
		},
	}
	manager.Rooms[permanentRoomCode] = &room

	manager.Tmpl = template.Must(template.ParseGlob("templates/*.html"))
	fs := http.FileServer(http.Dir("assets"))

	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	http.HandleFunc("/", manager.IndexHandler)
	http.HandleFunc("/room/", manager.RoomHandler)
	http.HandleFunc("/join", manager.JoinHandler)
	http.HandleFunc("/create", manager.CreateRoom)
	http.HandleFunc("/start", manager.StartGame)
	http.HandleFunc("/ws/chat", handlers.HandleChat)
	http.HandleFunc("/audio", handlers.HandleAudio)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v\n", sig)
		os.Exit(0)
	}()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("error listening:%v\n", err)
		return
	}
}
