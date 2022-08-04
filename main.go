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
		Conn:      conn,
		Pool:      pool,
		StopChan:  make(chan bool),
		WriteChan: make(chan []byte),
	}

	// it is OK to call this function for here because each endpoint is a goroutines
	// so the infinite for loop wont block the main thread
	client.Read()
}

func setupRoutes(pool *websocket.Pool) {

	go pool.Start()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(pool, w, r)
	})
}

func main() {
	PORT := "4444"
	fmt.Println("Server on listening port: " + PORT)

	pool := websocket.NewPool()
	defer pool.PowerOff()
	setupRoutes(pool)

	if err := http.ListenAndServe(":"+PORT, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
