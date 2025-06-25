package logic

import (
	"math/rand"
	"strings"
	"time"
)

func GenerateRoomCode(length int) string {
	const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var roomCode strings.Builder

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(len(characters))
		roomCode.WriteByte(characters[randomIndex])
	}

	return roomCode.String()
}
