package websocket

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID       string
	username string
	//in the future add an array for tags for groups and etc..
	Conn *websocket.Conn
	Pool *Pool
}

type Message struct {
	Type        int    `json:"type"`
	User        string `json:"user"`
	Content     string `json:"content"`
	Destinatary string `json:"destinatary"`
	Time        string `json:"time"`
}

type allUsersMessage struct {
	Type     int      `json:"type"`
	AllUsers []string `json:"allusers"`
	Time     string   `json:"time"`
}

func (c *Client) Read() {
	// this function is called if an error occurs and it just unregister from the pool and closes the connection
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	// LOGIN
	var firstmsg Message
	err := c.Conn.ReadJSON(&firstmsg)
	if err != nil {
		log.Println(err)
		return
	}
	//register to pool
	if firstmsg.Type == 2 {
		c.username = firstmsg.User
		c.Pool.Register <- c

	} else {
		fmt.Printf("client needs to first send username info")
		return
	}

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			return
		}

		// broadcast message to channel
		c.Pool.Broadcast <- msg
	}
}
