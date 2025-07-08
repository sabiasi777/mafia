package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sabiasi777/mafia/internal/handlers"
)

func main() {
	rm := handlers.NewRoomManager()

	rm.Tmpl = template.Must(template.ParseGlob("templates/*.html"))
	fs := http.FileServer(http.Dir("assets"))

	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	http.HandleFunc("/", rm.IndexHandler)
	http.HandleFunc("/room/", rm.RoomHandler)
	http.HandleFunc("/join", rm.JoinHandler)
	http.HandleFunc("/create", rm.CreateRoom)
	http.HandleFunc("/start", rm.StartGame)
	http.HandleFunc("/ws/chat", rm.HandleChat)

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
