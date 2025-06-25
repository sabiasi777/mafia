package handlers

import (
	"fmt"
	"net/http"
)

func (rm *RoomManager) IndexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("IndexHandler")
	if err := rm.Tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}
