package websocket

import (
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	username string
	password string
	//in the future add an array for tags for groups and etc..
	Conn      *websocket.Conn
	Pool      *Pool
	StopChan  chan bool
	WriteChan chan []byte
	mu        sync.Mutex
}

func (c *Client) Read() {
	// this function is called if an error occurs and it just unregister from the pool and closes the connection
	defer func() {

		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	// LOGIN
	var firstmsg RegisterMessage
	err := c.Conn.ReadJSON(&firstmsg)
	if err != nil {
		log.Println(err)
		return
	}
	//register to pool
	if firstmsg.Type == 2 {
		fmt.Println("received register messages")
		c.username = firstmsg.Username
		c.password = firstmsg.Password

		// after that i should start using mutex for safety
		c.Pool.Register <- c

	} else {
		fmt.Printf("client needs to first send username info")
		return
	}
	go c.writeMessages()
	for {

		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			return
		}
		// SendMsg Message to channel
		c.Pool.SendMsg <- msg
	}
}

func (c *Client) writeMessages() {
	for {
		select {
		case msg := <-c.WriteChan:
			if err := c.Conn.WriteMessage(1, msg); err != nil {
				c.mu.Lock()
				fmt.Println("could not send message to user: ", c.username, "because of ERROR::")
				c.mu.Unlock()
				fmt.Println(err)
				return
			}
		case a := <-c.StopChan:
			if a == true {
				fmt.Println("stoping because pool asked")
				return
			}

		}

	}
}
