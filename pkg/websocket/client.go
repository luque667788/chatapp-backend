package websocket

import (
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	username          string
	password          string
	Conn              *websocket.Conn
	Pool              *Pool
	StopChan          chan bool
	WriteChan         chan []byte
	mu                sync.Mutex
	forcedsocketclose bool
}

func (c *Client) Read() {
	// this function is called if an error occurs and it just unregister from the pool and closes the connection
	defer func() {
		if !c.forcedsocketclose {
			c.Pool.Unregister <- c
		}
		c.Conn.Close()
	}()
	c.forcedsocketclose = true

	var firstmsg RegisterMessage
	err := c.Conn.ReadJSON(&firstmsg)
	if err != nil {
		log.Println(err)
		return
	}
	if firstmsg.Type == 2 {
		fmt.Println("received first messages")
		c.username = firstmsg.Username
		c.password = firstmsg.Password

		c.Pool.Register <- c

	} else {
		fmt.Printf("client needs to first send username info")
		return
	}
	go c.writeMessages()
	select {
	case a := <-c.StopChan:
		if a {
			fmt.Println("stoping websocket connection because pool asked(1)")

			return
		}
	default:
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			return
		}
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
			if a {
				fmt.Println("stoping websocket connection because pool asked(2)")
				return
			}

		}

	}
}
