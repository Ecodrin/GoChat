package handlers

import (
	"net"
)

type Msg struct {
	Sender    string `json:"sender"`
	Receiver  string `json:"receiver"`
	Timestamp int64  `json:"timestamp"`
	Text      string `json:"text"`
	Status    int64  `json:"status"`
}

type AuthMsg struct {
	Login        string   `json:"login"`
	HashPassword [32]byte `json:"hash_password"`
	Timestamp    int64    `json:"timestamp"`
	Status       int64    `json:"status"`
}

type User struct {
	Login        string   `json:"login"`
	HashPassword [32]byte `json:"hash_password"`

	Conn   net.Conn
	Chats  map[string][]Msg
	Online bool
}
