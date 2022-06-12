package websocket

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

type RegisterMessage struct {
	Type     int    `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}
