package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Player struct {
	Name string
}

type Room struct {
	Code    string
	Players []Player
}

type Page struct {
	Title string
	Body  []byte
}

var rooms = make(map[string]*Room)
var tmpl *template.Template

// func changePlayers() { // room *Room
// 	f, err := os.Open("./templates/game.html")
// 	if err != nil {
// 		fmt.Printf("error opening file: %v\n", err)
// 		return
// 	}
// 	defer f.Close()

// 	doc, err := goquery.NewDocumentFromReader(f)
// 	if err != nil {
// 		fmt.Printf("Error creating document: %v\n", err)
// 		return
// 	}

// 	li := fmt.Sprintf("<li id=\"playerName\" >{{.Players[len(Players)-1].Name}}</li>")
// 	doc.Find("#playerList").AppendHtml(li)

// 	modifiedHtml, err := doc.Html()
// 	if err != nil {
// 		fmt.Printf("Error generating HTML: %v\n", err)
// 		return
// 	}
// 	fmt.Println(modifiedHtml)

// 	fWrite, err := os.Create("./templates/game.html")
// 	if err != nil {
// 		fmt.Printf("Error opening file for writing: %v\n", err)
// 		return
// 	}
// 	defer fWrite.Close()

// 	_, err = fWrite.WriteString(modifiedHtml)
// 	if err != nil {
// 		fmt.Printf("Error writing modified HTML to file %v:\n", err)
// 		return
// 	}
// }

func joinHandler(w http.ResponseWriter, r *http.Request) {
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

	room, exists := rooms[roomcode]
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	room.Players = append(room.Players, Player{Name: username})
	fmt.Println(room.Players)
	//changePlayers() // room

	http.Redirect(w, r, "/room/"+roomcode+"?user="+username, http.StatusSeeOther)
}

func createRoom(w http.ResponseWriter, r *http.Request) {
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
	roomCode := generateRoomCode(6)

	room := Room{
		Code:    roomCode,
		Players: []Player{},
	}

	room.Players = append(room.Players, Player{Name: username})
	rooms[roomCode] = &room
	fmt.Println("Room Created:", rooms[roomCode])
	//changePlayers() // &room

	http.Redirect(w, r, "/room/"+roomCode+"?user="+username, http.StatusSeeOther)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("IndexHandler")
	tmpl.ExecuteTemplate(w, "index.html", nil)
}

func roomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	roomCode := r.URL.Path[len("/room/"):]
	room, exists := rooms[roomCode]
	if !exists {
		http.NotFound(w, r)
		return
	}
	tmpl = template.Must(template.ParseGlob("./templates/game.html"))
	tmpl.ExecuteTemplate(w, "game.html", room)
}

func main() {
	tmpl = template.Must(template.ParseGlob("templates/*.html"))
	fs := http.FileServer(http.Dir("assets"))

	rooms["tSgK4H"] = &Room{Code: "tSgK4H", Players: []Player{}}

	http.Handle("/assets/", http.StripPrefix("/assets", fs))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/room/", roomHandler)
	http.HandleFunc("/join", joinHandler)
	http.HandleFunc("/create", createRoom)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v\n", sig)
		//removeTheLiElements()
		os.Exit(0)
	}()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("error listening:%v\n", err)
		return
	}

	//removeTheLiElements()
}

func removeTheLiElements() {
	f, err := os.Open("./templates/game.html")
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		return
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		fmt.Printf("Error creating document: %v\n", err)
		return
	}

	doc.Find("#playerName").Remove()
	updatedHtml, err := doc.Html()
	if err != nil {
		fmt.Printf("Error updating html: %v\n", err)
		return
	}

	fWrite, err := os.Create("./templates/game.html")
	if err != nil {
		fmt.Printf("Error opening file for writing: %v\n", err)
		return
	}
	defer fWrite.Close()

	_, err = fWrite.WriteString(updatedHtml)
	if err != nil {
		fmt.Printf("Error writing modified HTML to file %v:\n", err)
		return
	}

	fmt.Println("Elements have been removed successfuly")
}

func generateRoomCode(length int) string {
	const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var roomCode strings.Builder

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(len(characters))
		roomCode.WriteByte(characters[randomIndex])
	}

	return roomCode.String()
}
