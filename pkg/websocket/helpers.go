package websocket

import (
	"encoding/json"
	"log"

	"github.com/go-redis/redis/v8"
)

func publishMessage(conn *redis.Client, payload []byte, channel string) {

	err2 := conn.Publish(ctx, channel, payload).Err()

	if err2 != nil {
		log.Println(err2)
		panic(err2)
	}
}

func GetAllUsers(conn *redis.Client) []string {
	//gets all users including the recently added user
	val, err := conn.Do(ctx, "SMEMBERS", "clients").Result()
	if err != nil {
		panic(err)
	}

	allusers := make([]string, len(val.([]interface{})))
	for p, value := range val.([]interface{}) {
		allusers[p] = value.(string)
	}
	return allusers
}

func CheckUserExist(conn *redis.Client, username string) bool {
	result, err := conn.Do(ctx, "EXISTS", username).Int()
	if err != nil {
		panic(err)
	} else if result == 1 {
		return true
	}
	return false
}

func RemoveUserRedis(conn *redis.Client, username string) {
	_, err := conn.Do(ctx, "DEL", username).Result()
	if err != nil {
		panic(err)
	}
	_, err = conn.Do(ctx, "SREM", "clients", username).Result()
	if err != nil {
		panic(err)
	}
}

func UpdateClientsList(conn *redis.Client) {
	allusers := GetAllUsers(conn)
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
	publishMessage(conn, payload, registerChannel)
}
