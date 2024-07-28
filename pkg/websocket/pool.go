package websocket

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var channelGeneral = "GENERAL"
var registerChannel = "REGISTER"

// we need a pool to avoid concurrent acess to the Clients map ( and also secundarily to SendMsg a Message)
type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	Clients    map[string]*Client
	SendMsg    chan Message
	Redis      *redis.Client
	name       string
	mu         sync.Mutex
	redismu    sync.Mutex
}

func NewPool() *Pool {
	//maybe add dependency injection here in the future not very urgent
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[string]*Client),
		SendMsg:    make(chan Message),
		Redis: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}),
		name: "",
	}
}
func (pool *Pool) subscribeToMessages() {
	pool.redismu.Lock()
	pubsub := pool.Redis.Subscribe(ctx, pool.name, registerChannel)
	pool.redismu.Unlock()
	// Close the subscription when we are done.
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			panic(err)
		}

		switch msg.Channel {
		case registerChannel:
			pool.mu.Lock()
			if len(pool.Clients) == 0 {
				fmt.Println("No clients in the pool to send the message to.")
				pool.mu.Unlock()
				break
			}
			fmt.Println("--|sending  to all users on " + pool.name + "  the updated the users list|")
			for _, _client := range pool.Clients {
				_client.WriteChan <- []byte(msg.Payload)

			}
			pool.mu.Unlock()

		case pool.name:
			//decode Message
			var Message = DecodeJson([]byte(msg.Payload))

			fmt.Println("Sending Message to:" + Message.Destinatary + " from: " + Message.User)
			pool.mu.Lock()
			pool.Clients[Message.Destinatary].WriteChan <- []byte(msg.Payload)
			pool.mu.Unlock()

		}

	}

}
func (pool *Pool) Start() {
	pool.redismu.Lock()
	val, err := pool.Redis.Do(ctx, "INCR", "server:number").Result()
	pool.redismu.Unlock()
	if err != nil {
		panic(err)
	}
	pool.mu.Lock()
	pool.name = fmt.Sprint("server:", val)
	pool.mu.Unlock()

	fmt.Println("name " + pool.name)
	go pool.subscribeToMessages()
	for {
		select {
		case client := <-pool.Register:
			client.mu.Lock()
			if CheckUserHashExist(pool.Redis, &pool.mu, client.username) {
				// check if user is already online
				if CheckUserOnline(pool.Redis, &pool.redismu, client.username) {
					fmt.Println("user: ", client.username, "is already online but try to login")
					client.StopChan <- true
					client.mu.Unlock()
					break
				}

				//check if password is correct
				err := VerifyPassword(GetUserHashItem(pool.Redis, &pool.redismu, client.username, "password"), client.password)
				if err != nil {
					client.StopChan <- true
					fmt.Println("incorrect password for username: ", client.username)
					client.mu.Unlock()
					break
				} else {
					pool.mu.Lock()
					fmt.Println("user", client.username, "will LOGIN at", pool.name)
					SetUserHashItem(pool.Redis, &pool.redismu, client.username, "server", pool.name)
					pool.mu.Unlock()
					ActivateUserRedis(pool.Redis, &pool.redismu, client.username, pool.name)
					pool.mu.Lock()
					pool.Clients[client.username] = client
					UpdateRedisClientsList(pool.Redis, &pool.redismu)
					pool.mu.Unlock()
					SendPreviousMessages(pool.Redis, &pool.redismu, client)

					client.forcedsocketclose = false
					client.mu.Unlock()
					break

				}

			} else {
				fmt.Println("user", client.username, "will be SIGNUP at", pool.name)
				Hash, err := Hash(client.password)
				if err != nil {
					panic(err)
				}
				client.password = string(Hash)
				pool.mu.Lock()
				AddUserRedis(pool.Redis, &pool.redismu, pool.name, client.username)
				pool.mu.Unlock()
				SetUserHashItem(pool.Redis, &pool.redismu, client.username, "password", client.password)
				ActivateUserRedis(pool.Redis, &pool.redismu, client.username, pool.name)
				pool.mu.Lock()
				pool.Clients[client.username] = client
				pool.mu.Unlock()
				UpdateRedisClientsList(pool.Redis, &pool.redismu)
				client.forcedsocketclose = false
				client.mu.Unlock()
				break
			}

		case client := <-pool.Unregister:
			client.mu.Lock()
			if !CheckUserHashExist(pool.Redis, &pool.redismu, client.username) {
				fmt.Println("user: ", client.username, "do not exist but tries to unregister")
				client.mu.Unlock()
				break
			}
			if !CheckUserOnline(pool.Redis, &pool.redismu, client.username) {
				fmt.Println("-------------------->user ", client.username, " was rejected and is diconnecting")
				client.mu.Unlock()
				break
			}
			DeactivateUserRedis(pool.Redis, &pool.redismu, client.username)
			fmt.Println("user", client.username, "disconnected ")
			pool.mu.Lock()
			delete(pool.Clients, client.username)
			pool.mu.Unlock()
			client.mu.Unlock()
			break

		//received a Message
		case Message := <-pool.SendMsg:
			if !CheckUserHashExist(pool.Redis, &pool.redismu, Message.Destinatary) {
				fmt.Println("user: ", Message.Destinatary, "do NOT EXIST but tries to receive message")
				break
			}
			if !CheckUserHashExist(pool.Redis, &pool.redismu, Message.User) {
				fmt.Println("user: ", Message.User, "do NOT EXIST but tries to send message")
				break
			}
			if !CheckUserOnline(pool.Redis, &pool.redismu, Message.User) {
				fmt.Println(" user: ", Message.Destinatary, " is NOT ONLINE but tries to SEND message")
				break
			}
			if !CheckUserOnline(pool.Redis, &pool.redismu, Message.Destinatary) {
				fmt.Println(" user: ", Message.Destinatary, "is NOT ONLINE, archiving the messages for later sending when avaible")
				pool.mu.Lock()
				pool.Clients[Message.User].WriteChan <- []byte(EncodeJson(Message))
				pool.mu.Unlock()
				SaveMessageDB(pool.Redis, &pool.redismu, Message)
				break
			}

			/*

				-each user has a set with conversation he participates

				-after sending a message we archive the message to a set with all messages of a conversation

				-the destinatary and sender of the message conversation list sets will be added
				(or overwritten rewritten) with the messages conversation
				- only one redis set will be used for each conversation so:
				->>>the name of the chat will be the names of the users in alphabetical order like "alice:joao"

			*/

			//sends Message to user who sent the Message
			pool.mu.Lock()
			pool.Clients[Message.User].WriteChan <- []byte(EncodeJson(Message))

			//sends Message to user who will receive Message (it may be in other server (horizontal scaling))
			publishMessage(pool.Redis, &pool.redismu, EncodeJson(Message), GetUserServerRedis(pool.Redis, &pool.redismu, Message.Destinatary))
			pool.mu.Unlock()
			SaveMessageDB(pool.Redis, &pool.redismu, Message)
			break

		}
	}
}

func (pool *Pool) PowerOff() {
	fmt.Println(pool.name + " powering off")
	pool.mu.Lock()
	for _, client := range pool.Clients {
		client.mu.Lock()

		UpdateRedisClientsList(pool.Redis, &pool.redismu)
		fmt.Println("user", client.username, "disconnected ")
		DeactivateUserRedis(pool.Redis, &pool.redismu, client.username)
		delete(pool.Clients, client.username)
		client.mu.Unlock()
	}
	//TODO! delete the server from the list of servers
	pool.mu.Unlock()

}
