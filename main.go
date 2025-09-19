package main

import (
	"context"
	"fmt"
	"net/http"
)

var port string = "8000"

func main() {
	setUpApi()
	fmt.Printf("Server started at port: %n \n ", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		return
	}
}

func setUpApi() {
	ctx := context.Background()
	manager := NewManager(ctx)
	http.Handle("/", http.FileServer(http.Dir("public")))
	http.HandleFunc("/ws", manager.ServeWS)
	http.HandleFunc("/rooms", manager.GetRooms)
}
