package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var channelGeneral = "GENERAL"
var registerChannel = "REGISTER"

func (pool *Pool) publishMessage(payload []byte, channel string) {

	err2 := pool.Redis.Publish(ctx, channel, payload).Err()

	if err2 != nil {
		log.Println(err2)
		panic(err2)
	}
}

func (pool *Pool) subscribeToMessages() {

	pubsub := pool.Redis.Subscribe(ctx, pool.name, registerChannel)

	// Close the subscription when we are done.
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			panic(err)
		}

		switch msg.Channel {
		case registerChannel:

			fmt.Println("--|sending  to all users on " + pool.name + "  the updated the users list|")
			for _, _client := range pool.Clients {
				_client.Conn.WriteMessage(1, []byte(msg.Payload))

			}

		case pool.name:
			//decode message
			var message Message
			if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
				panic(err)
			}

			fmt.Println("Sending message to:" + message.Destinatary + " from: " + message.User)
			if err := pool.Clients[message.Destinatary].Conn.WriteMessage(1, []byte(msg.Payload)); err != nil {
				fmt.Println(err)
				return
			}

		}

	}

}

// we need a pool to avoid concurrent acess to the Clients map ( and also secundarily to broadcast a message)
type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	Clients    map[string]*Client
	Broadcast  chan Message
	Redis      *redis.Client
	name       string
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[string]*Client),
		Broadcast:  make(chan Message),
		Redis: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}),
		name: "",
	}
}

func GetAllUsers(conn *redis.Client) []string {
	//gets all users including the recently added user
	val, err := conn.Do(ctx, "HKEYS", "clients").Result()
	if err != nil {
		panic(err)
	}

	allusers := make([]string, len(val.([]interface{})))
	for p, value := range val.([]interface{}) {
		allusers[p] = value.(string)
	}
	return allusers
}

func (pool *Pool) Start() {
	val, err := pool.Redis.Do(ctx, "INCR", "server:number").Result()
	if err != nil {
		panic(err)
	}
	pool.name = fmt.Sprint("server:", val)

	fmt.Println("name " + pool.name)
	go pool.subscribeToMessages()
	for {
		select {
		//register in the pool
		case client := <-pool.Register:
			fmt.Println("user", client.username, "will be register at", pool.name)

			//add user to redis all users hash
			_, err := pool.Redis.Do(ctx, "HSET", "clients", client.username, pool.name).Result()
			if err != nil {
				panic(err)
			}

			allusers := GetAllUsers(pool.Redis)
			// transform it to json
			var allusersjson allUsersMessage = allUsersMessage{
				Type:     3,
				AllUsers: allusers,
			}
			payload, err := json.Marshal(allusersjson)
			if err != nil {
				panic(err)
			}
			//publish to redis
			//in the future make a better way of updating user list in front-end
			pool.publishMessage(payload, registerChannel)

			//add to local map of users in the server
			pool.Clients[client.username] = client
			break
		// unregister from the pool
		case client := <-pool.Unregister:
			_, err := pool.Redis.Do(ctx, "HDEL", "clients", client.username).Result()
			if err != nil {
				panic(err)
			}

			allusers := GetAllUsers(pool.Redis)
			// transform it to json
			var allusersjson allUsersMessage = allUsersMessage{
				Type:     3,
				AllUsers: allusers,
			}
			payload, err := json.Marshal(allusersjson)
			if err != nil {
				panic(err)
			}
			//publish to redis
			pool.publishMessage(payload, registerChannel)
			delete(pool.Clients, client.username)
			fmt.Println("user disconnected ")
			break
		//received a message
		case message := <-pool.Broadcast:
			//send message to user who sent the message
			if err := pool.Clients[message.User].Conn.WriteJSON(message); err != nil {
				fmt.Println(err)
				return
			}
			payload, err := json.Marshal(message)
			if err != nil {
				panic(err)
			}

			server, err := pool.Redis.Do(ctx, "HGET", "clients", message.Destinatary).Text()
			if err != nil {
				if err == redis.Nil {
					fmt.Println("user not found")
					break
				} else {
					panic(err)
				}
			}
			//send message to user wo will receive message (it may be in other server)
			pool.publishMessage(payload, server)

		}
	}
}

func (pool *Pool) PowerOff() {
	// properly warn other servers that this server is turning off
	//need to implemenet a way of transfering clients
	// or implement a way of warining client that the server was powered off
	//and maybe gray out the client on the front end
	fmt.Println(pool.name + " powering off")
	for _, client := range pool.Clients {
		_, err := pool.Redis.Do(ctx, "HDEL", "clients", client.username).Result()
		if err != nil {
			panic(err)
		}

		allusers := GetAllUsers(pool.Redis)
		// transform it to json
		var allusersjson allUsersMessage = allUsersMessage{
			Type:     3,
			AllUsers: allusers,
		}
		payload, err := json.Marshal(allusersjson)
		if err != nil {
			panic(err)
		}
		//publish to redis
		pool.publishMessage(payload, registerChannel)
		fmt.Println("user " + client.username + " disconnected ")
		delete(pool.Clients, client.username)

	}

}
