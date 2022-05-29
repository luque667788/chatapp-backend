package main

import (
	"chatapp/pkg/websocket"
	"fmt"
	"log"
	"net/http"
)

func serveWs(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	client := &websocket.Client{
		Conn: conn,
		Pool: pool,
	}

	// it is OK to call this function for here because each endpoint is a goroutines
	// so the infinite for loop wont block the main thread
	client.Read()
}

func setupRoutes() {
	pool := websocket.NewPool()
	go pool.Start()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(pool, w, r)
	})
}

func main() {

	fmt.Println("Distributed Chat App v0.01")
	setupRoutes()
	if err := http.ListenAndServe(":3032", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
