package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/bcrypt"
)

func publishMessage(conn *redis.Client, mu *sync.Mutex, payload []byte, channel string) {
	mu.Lock()
	err2 := conn.Publish(ctx, channel, payload).Err()
	mu.Unlock()

	if err2 != nil {
		log.Println(err2)
		panic(err2)
	}
}

func GetAllUsers(conn *redis.Client, mu *sync.Mutex) []string {
	mu.Lock()
	val, err := conn.Do(ctx, "SMEMBERS", "clients:online").Result()
	mu.Unlock()
	if err != nil {
		panic(err)
	}

	allusers := make([]string, len(val.([]interface{})))
	for p, value := range val.([]interface{}) {
		allusers[p] = value.(string)
	}
	return allusers
}

func GetAllRegisteredUsers(conn *redis.Client, mu *sync.Mutex) []string {
	mu.Lock()
	val, err := conn.Do(ctx, "SMEMBERS", "clients").Result()
	mu.Unlock()
	if err != nil {
		panic(err)
	}

	allusers := make([]string, len(val.([]interface{})))
	for p, value := range val.([]interface{}) {
		allusers[p] = value.(string)
	}
	return allusers
}

func CheckUserHashExist(conn *redis.Client, mu *sync.Mutex, username string) bool {
	mu.Lock()
	result, err := conn.Do(ctx, "EXISTS", username).Int()
	mu.Unlock()
	if err != nil {
		panic(err)
	} else if result == 1 {
		return true
	}
	return false
}

func RemoveUserRedis(conn *redis.Client, mu *sync.Mutex, username string) {
	mu.Lock()
	_, err := conn.Do(ctx, "DEL", username).Result()
	if err != nil {
		panic(err)
	}
	_, err = conn.Do(ctx, "SREM", "clients", username).Result()
	if err != nil {
		panic(err)
	}
	mu.Unlock()
}

func DeactivateUserRedis(conn *redis.Client, mu *sync.Mutex, username string) {

	fmt.Println("user", username, "is being deactivated ")
	mu.Lock()
	_, err := conn.Do(ctx, "SREM", "clients:online", username).Result()
	mu.Unlock()
	if err != nil {
		panic(err)
	}
	mu.Lock()
	_, err2 := conn.Do(ctx, "HSET", username, "server", "notdefined").Result()
	mu.Unlock()
	if err2 != nil {
		panic(err2)
	}
}

func ActivateUserRedis(conn *redis.Client, mu *sync.Mutex, username string, poolname string) {
	mu.Lock()
	_, err := conn.Do(ctx, "SADD", "clients:online", username).Result()
	mu.Unlock()
	if err != nil {
		panic(err)
	}
	mu.Lock()

	_, err2 := conn.Do(ctx, "HSET", username, "server", poolname).Result()
	mu.Unlock()
	if err2 != nil {
		panic(err2)
	}
}

func CheckUserOnline(conn *redis.Client, mu *sync.Mutex, username string) bool {
	mu.Lock()
	result, err := conn.Do(ctx, "SISMEMBER", "clients:online", username).Int()
	mu.Unlock()
	if err != nil {
		panic(err)
	} else if result == 1 {
		return true
	}
	return false
}

func GetUserHashItem(conn *redis.Client, mu *sync.Mutex, username string, key string) string {
	mu.Lock()
	info, err := conn.Do(ctx, "HGET", username, key).Text()
	mu.Unlock()
	if err != nil {
		panic(err)
	}
	return info
}

func SetUserHashItem(conn *redis.Client, mu *sync.Mutex, username string, key string, value string) {
	mu.Lock()
	err := conn.HSet(ctx, username, key, value).Err()
	mu.Unlock()
	if err != nil {
		fmt.Println(value)
		fmt.Println(key)
		fmt.Println(username)
		fmt.Println("ERROR setting ", key, " from user: ", username)
		panic(err)
	}
}

func AddUserRedis(conn *redis.Client, mu *sync.Mutex, poolname string, username string) {
	mu.Lock()
	_, errr := conn.Do(ctx, "SADD", "clients", username).Result()
	mu.Unlock()
	if errr != nil {
		panic(errr)
	}
}

func UpdateRedisClientsList(conn *redis.Client, mu *sync.Mutex) {
	allusers := GetAllRegisteredUsers(conn, mu)
	var allusersjson allUsersMessage = allUsersMessage{
		Type:     3,
		AllUsers: allusers,
	}
	fmt.Println(allusers)
	payload, err := json.Marshal(allusersjson)
	if err != nil {
		panic(err)
	}
	publishMessage(conn, mu, payload, registerChannel)
}

func Hash(password string) ([]byte, error) {
	value, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return value, err
}

func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func EncodeJson(message Message) []byte {
	payload, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	return payload
}

func DecodeJson(payload []byte) Message {
	var Message Message
	if err := json.Unmarshal([]byte(payload), &Message); err != nil {
		panic(err)
	}
	return Message
}

func GetUserServerRedis(conn *redis.Client, mu *sync.Mutex, username string) string {

	return GetUserHashItem(conn, mu, username, "server")
}

func SaveMessageDB(conn *redis.Client, mu *sync.Mutex, msg Message) {
	room := CompareUsername(msg.User, msg.Destinatary)
	mu.Lock()
	_, err := conn.Do(ctx, "LPUSH", room, string(EncodeJson(msg))).Result()
	mu.Unlock()
	if err != nil {
		panic(err)
	}
}

func SendPreviousMessages(conn *redis.Client, mu *sync.Mutex, client *Client) {
	mu.Lock()
	val, err := conn.Do(ctx, "SMEMBERS", "clients").Result()
	mu.Unlock()
	if err != nil {
		panic(err)
	}

	allusers := make([]string, len(val.([]interface{})))
	for p, value := range val.([]interface{}) {
		allusers[p] = value.(string)
	}
	for _, name := range allusers {
		if name != client.username {

			room := CompareUsername(client.username, name)
			mu.Lock()
			val2, err2 := conn.Do(ctx, "LRANGE", room, "0", "-1").Result()
			mu.Unlock()
			if err2 != nil {
				panic(err2)
			}
			users := make([]string, len(val2.([]interface{})))
			for p := 0; p < len(val2.([]interface{})); p++ {
				users[p] = val2.([]interface{})[p].(string)
			}
			for i := len(users) - 1; i >= 0; i-- {
				fmt.Println("sending messages that were sent earlier")
				client.WriteChan <- []byte(users[i])
			}

		}
	}

}

// er -> e
func CompareUsername(user1 string, user2 string) string {
	if []rune(user1)[0] > []rune(user2)[0] {
		return fmt.Sprint(user1, ":", user2)
	} else if []rune(user1)[0] < []rune(user2)[0] {
		return fmt.Sprint(user2, ":", user1)
	} else if []rune(user1)[0] == []rune(user2)[0] {
		i := 1
		for {
			if i == len(user1) {
				return fmt.Sprint(user1, ":", user2)
			} else if i == len(user2) {
				return fmt.Sprint(user2, ":", user1)
			}
			if []rune(user1)[i] > []rune(user2)[i] {
				return fmt.Sprint(user1, ":", user2)
			} else if []rune(user1)[i] < []rune(user2)[i] {
				return fmt.Sprint(user2, ":", user1)
			} else if []rune(user1)[i] == []rune(user2)[i] {
				i += 1
			}
		}
	}
	panic(fmt.Sprint("problem comparing usernames", user1, "  ", user2))
}

func senderrortoclient(c *Client, errorclientmsg string) {
	var msge ErrorMessage = ErrorMessage{
		Type:    4,
		Content: errorclientmsg,
	}

	payload, err := json.Marshal(msge)
	if err != nil {
		panic(err)
	}
	if err := c.Conn.WriteMessage(1, []byte(payload)); err != nil {

		fmt.Println("could not send CONNECTION STATUS to user: ", c.username, "because of ERROR::")
		fmt.Println(err)
	}

}
