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
		fmt.Println("closing whole connection of user: ", c.username)
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
	var wg = 0

	go func() {
		defer func() {
			fmt.Println("CLOSE the writing channel ", c.username)
			wg = wg + 1
		}()
		c.writeMessages()
	}()
	go func() {
		defer func() {
			fmt.Println("CLOSE the reading channel ", c.username)
			wg = wg + 1
		}()
		c.readMessages()
	}()

	for {
		if wg > 0 {
			break
		}
	}

}

func (c *Client) readMessages() {
	for {
		var msg Message

		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			fmt.Println("could not RECEIVE message to user: ", c.username, "because of ERROR::")
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
				fmt.Println("stoping websocket connection because pool asked")
				return
			}

		}

	}
}
