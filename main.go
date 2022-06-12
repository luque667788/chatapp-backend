package main

import (
	"chatapp/pkg/websocket"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	PORT := os.Args[1:]
	fmt.Println("Server on listening port: " + PORT[0])

	pool := websocket.NewPool()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			// sig is a ^C, handle it
			log.Printf("captured %v, stopping profiler and exiting..", sig)
			pool.PowerOff()
			os.Exit(1)
		}
	}()
	defer pool.PowerOff()
	setupRoutes(pool)

	if err := http.ListenAndServe(":"+PORT[0], nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
