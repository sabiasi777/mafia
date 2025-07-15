package logic

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/sabiasi777/mafia/internal/models"
)

func AssignRoles(room *models.Room) {
	numPlayers := len(room.Players)
	roles := []string{}

	switch {
	case numPlayers == 2:
		roles = []string{"Mafia", "Doctor", "Villager", "Villager"}
	case numPlayers <= 6:
		roles = []string{"Mafia", "Doctor", "Detective", "Villager", "Villager", "Villager"}
	case numPlayers <= 8:
		roles = []string{"Mafia", "Mafia", "Doctor", "Detective", "Villager", "Villager", "Villager", "Villager"}
	case numPlayers <= 10:
		roles = []string{"Mafia", "Mafia", "Doctor", "Detective", "Bodyguard", "Villager", "Villager", "Villager", "Villager", "Villager"}
	default:
		roles = []string{"Mafia", "Mafia", "Godfather", "Doctor", "Detective", "Bodyguard"}
		for len(roles) < numPlayers {
			roles = append(roles, "Villager")
		}
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(roles), func(i, j int) {
		roles[i], roles[j] = roles[j], roles[i]
	})

	for i := range room.Players {
		room.Players[i].Role = roles[i]
	}

}

func GetActiveRoles(playerCount int) []string {
	fmt.Println("PlayerCount in GetActiveRoles:", playerCount)

	switch playerCount {
	case 2:
		return []string{"Mafia", "Villager"}
	case 3:
		return []string{"Mafia", "Doctor", "Villager"}
	case 4:
		return []string{"Mafia", "Doctor", "Detective", "Villager"}
	case 5:
		return []string{"Mafia", "Doctor", "Detective", "Villager", "Villager"}
	case 6:
		return []string{"Mafia", "Doctor", "Detective", "Bodyguard", "Villager", "Villager"}
	case 7:
		return []string{"Mafia", "Mafia", "Doctor", "Detective", "Bodyguard", "Villager", "Villager"}
	case 8:
		return []string{"Mafia", "Mafia", "Doctor", "Detective", "Bodyguard", "Villager", "Villager", "Villager"}
	// Add cases for 9, 10, etc., as needed

	default:
		return []string{"No roles yet"}
	}
}
