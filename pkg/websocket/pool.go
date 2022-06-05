package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var channelGeneral = "GENERAL"
var registerChannel = "REGISTER"

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
	//maybe add dependency injection here in the future not very urgent
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
			if CheckUserExist(pool.Redis, client.username) {
				client.UnregisterChan <- true
				fmt.Println("user: ", client.username, " already exists")
				break
			}
			fmt.Println("user", client.username, "will be register at", pool.name)

			//add user password
			_, err := pool.Redis.Do(ctx, "HSET", client.username, "server", pool.name).Result()
			if err != nil {
				panic(err)
			}
			_, errr := pool.Redis.Do(ctx, "SADD", "clients", client.username).Result()
			if errr != nil {
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
			client.UnregisterChan <- false
			publishMessage(pool.Redis, payload, registerChannel)

			//add to local map of users in the server
			pool.Clients[client.username] = client

			break
		// unregister from the pool
		case client := <-pool.Unregister:
			if CheckUserExist(pool.Redis, client.username) == false {
				fmt.Println("user: ", client.username, "do not exist but tries to unregister")
				break
			}
			_, err := pool.Redis.Do(ctx, "DEL", client.username).Result()
			if err != nil {
				panic(err)
			}
			_, err = pool.Redis.Do(ctx, "SREM", "clients", client.username).Result()
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
			publishMessage(pool.Redis, payload, registerChannel)
			fmt.Println("user", client.username, "disconnected ")
			delete(pool.Clients, client.username)

			break
		//received a message
		case message := <-pool.Broadcast:
			if CheckUserExist(pool.Redis, message.Destinatary) == false {
				fmt.Println("user: ", message.Destinatary, "do not exist")
				break
			}
			if CheckUserExist(pool.Redis, message.User) == false {
				fmt.Println("user: ", message.User, "do not exist But something is very wrong")
				panic("---- ERROR ----- user do not exist but sent a message")
			}

			//send message to user who sent the message
			if err := pool.Clients[message.User].Conn.WriteJSON(message); err != nil {
				fmt.Println(err)
				return
			}
			payload, err := json.Marshal(message)
			if err != nil {
				panic(err)
			}

			server, err := pool.Redis.Do(ctx, "HGET", message.Destinatary, "server").Text()
			if err != nil {
				panic(err)
			}
			//send message to user wo will receive message (it may be in other server)
			publishMessage(pool.Redis, payload, server)

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
		RemoveUserRedis(pool.Redis, client.username)

		UpdateClientsList(pool.Redis)
		fmt.Println("user", client.username, "disconnected ")
		delete(pool.Clients, client.username)
	}

}
