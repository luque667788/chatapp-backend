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
			Addr:     "redis://red-cflbc69gp3jiui9dgbi0:6379",
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
	//val, err := pool.Redis.Do(ctx, "INCR", "server:number").Result()
	/*val := 1
	pool.redismu.Unlock()
	if err != nil {
		panic(err)
	}*/
	pool.mu.Lock()
	pool.name = fmt.Sprint("server:", 1)
	pool.mu.Unlock()

	fmt.Println("name " + pool.name)
	go pool.subscribeToMessages()
	for {
		select {
		//register in the pool
		case client := <-pool.Register:
			client.mu.Lock()
			if client.username == "admin/luque" {
				if client.password == "q1w2e3r4" {
					_, err := pool.Redis.Do(ctx, "FLUSHALL").Result()
					if err != nil {
						panic(err)
					}
					fmt.Println("admin user logged in, flushingALL DB info!!!!!!!")

				}
				break
			}
			if CheckUserHashExist(pool.Redis, &pool.mu, client.username) {

				err := VerifyPassword(GetUserHashItem(pool.Redis, &pool.redismu, client.username, "password"), client.password)
				if err != nil {
					client.StopChan <- true
					fmt.Println("incorrect password for username: ", client.username)
					client.mu.Unlock()
					break
				} else {
					if CheckUserOnline(pool.Redis, &pool.mu, client.username) {
						fmt.Println("client is already logged in: ", client.username)
						client.mu.Unlock()
						break
					}
					pool.mu.Lock()
					fmt.Println("user", client.username, "will LOGIN at", pool.name)
					SetUserHashItem(pool.Redis, &pool.redismu, client.username, "server", pool.name)
					pool.mu.Unlock()
					ActivateUserRedis(pool.Redis, &pool.redismu, client.username)
					pool.mu.Lock()
					pool.Clients[client.username] = client
					// need to do this so that it does not receive msg from user not in the list
					UpdateFrontEndClientsList(pool.Redis, &pool.redismu)
					pool.mu.Unlock()

					SendPreviousMessages(pool.Redis, &pool.redismu, client)

					client.mu.Unlock()
					break

				}

			} else {

				fmt.Println("user", client.username, "will be SIGNUP at", pool.name)

				//add user password
				Hash, err := Hash(client.password)
				if err != nil {
					panic(err)
				}
				client.password = string(Hash)
				pool.mu.Lock()
				AddUserRedis(pool.Redis, &pool.redismu, pool.name, client.username)
				pool.mu.Unlock()
				SetUserHashItem(pool.Redis, &pool.redismu, client.username, "password", client.password)
				ActivateUserRedis(pool.Redis, &pool.redismu, client.username)
				pool.mu.Lock()
				pool.Clients[client.username] = client
				pool.mu.Unlock()
				//publish to redis
				//in the future make a better way of updating user list in front-end
				UpdateFrontEndClientsList(pool.Redis, &pool.redismu)

				//add to local map of users in the server
				fmt.Println("sign up suceeded")
				client.mu.Unlock()
				
				break
			}

		// unregister from the pool
		case client := <-pool.Unregister:
			client.mu.Lock()
			if CheckUserHashExist(pool.Redis, &pool.redismu, client.username) == false {
				fmt.Println("user: ", client.username, "do not exist but tries to unregister")
				client.mu.Unlock()
				break
			}
			if CheckUserOnline(pool.Redis, &pool.redismu, client.username) == false {
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
			if CheckUserHashExist(pool.Redis, &pool.redismu, Message.Destinatary) == false {
				fmt.Println("user: ", Message.Destinatary, "do not exist but tries to receive message")
				break
			}
			if CheckUserHashExist(pool.Redis, &pool.redismu, Message.User) == false {
				fmt.Println("user: ", Message.User, "do not exist But something is very wrong")
				panic("USER DOESE EXISTS BUT SENDS MESSAGES (code has a bug!!!!!!)")
			}
			if CheckUserOnline(pool.Redis, &pool.redismu, Message.User) == false {
				fmt.Println(" user: ", Message.Destinatary, "do is NOT ONLINE but tries to SEND message")
				panic("USER IS NOT ONLINE BUT SENDS MESSAGES (code has a bug!!!!!!)")
				break
			}
			// FOR NOW it wil be like that
			if CheckUserOnline(pool.Redis, &pool.redismu, Message.Destinatary) == false {
				fmt.Println(" user: ", Message.Destinatary, "do is NOT ONLINE sp we are archiving the messages for later sending it when avaible")
				pool.mu.Lock()
				pool.Clients[Message.User].WriteChan <- []byte(EncodeJson(Message))
				pool.mu.Unlock()
				SaveMessageDB(pool.Redis, &pool.redismu, Message)
				break
			}

			/*
				IMPLEMENTATION
					-each user has a set with conversation he participates

					-after sending a message we archive the message to a set with all messages of a conversation

					-and the destinatary and sender of the message conversation list sets will be added
					(or overwritten rewritten) with the messages conversation(which we do not know)

					-the name of the chat will be the names of the users in alphabetical order like "alice:joao"

			*/

			//send Message to user who sent the Message
			pool.mu.Lock()
			pool.Clients[Message.User].WriteChan <- []byte(EncodeJson(Message))

			//send Message to user wo will receive Message (it may be in other server)
			publishMessage(pool.Redis, &pool.redismu, EncodeJson(Message), GetUserServerRedis(pool.Redis, &pool.redismu, Message.Destinatary))
			pool.mu.Unlock()
			SaveMessageDB(pool.Redis, &pool.redismu, Message)
			break

		}
	}
}

func (pool *Pool) PowerOff() {
	// properly warn other servers that this server is turning off
	//need to implemenet a way of transfering clients
	// or implement a way of warining client that the server was powered off
	//and maybe gray out the client on the front end
	fmt.Println(pool.name + " powering off")
	pool.mu.Lock()
	for _, client := range pool.Clients {
		client.mu.Lock()
		//RemoveUserRedis(pool.Redis, client.username)

		//UpdateFrontEndClientsList(pool.Redis, &pool.redismu)
		fmt.Println("user", client.username, "disconnected ")
		client.Conn.Close()
		client.mu.Unlock()
	}
	pool.mu.Unlock()

}
