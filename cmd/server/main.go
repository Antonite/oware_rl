package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Antonite/oware_rl/server"
)

func main() {
	server := server.New()

	http.HandleFunc("/moves", func(w http.ResponseWriter, r *http.Request) {
		server.GetMovesHandler(w, r)
	})

	http.HandleFunc("/board", func(w http.ResponseWriter, r *http.Request) {
		server.GetBoardHandler(w, r)
	})

	fmt.Println("Server started   " + time.Now().Format("Mon Jan _2 15:04:05 2006"))
	log.Fatal(http.ListenAndServe(":8081", nil))
}
