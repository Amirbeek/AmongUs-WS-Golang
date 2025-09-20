package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

func main() {
	setupAPI()
	fmt.Print("Server started at port 3000\n")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func setupAPI() {
	ctx := context.Background()
	manager := NewManager(ctx)

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/ws", manager.ServeWS)
	http.HandleFunc("/rooms", manager.getRooms)
}
