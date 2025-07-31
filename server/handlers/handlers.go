package handlers

import (
	"net"
	"time"
)


type Conversation struct {
    ID        	int       	`json:"id"`
	SenderId 	int 		`json:"sender_id"`
	ReceiverId 	int 		`json:"receiver_id"`
    CreatedAt 	time.Time 	`json:"created_at"`
}

type Msg struct {
	ID			int 		`json:"id"`
	Sender    	string 		`json:"sender"`
	Receiver  	string 		`json:"receiver"`
	Timestamp 	int64  		`json:"timestamp"`
	Text      	string 		`json:"text"`
	Status    	int64  		`json:"status"`

}

type DataBaseMsg struct {
	ID             int       `json:"id"`
    ConversationId int       `json:"conversation_id"`
    SenderId       int       `json:"sender_id"`
    Body           string    `json:"body"`
    SentAt         time.Time `json:"sent_at"`
}

type AuthMsg struct {
	Login        string   `json:"login"`
	HashPassword [32]byte `json:"hash_password"`
	Timestamp    int64    `json:"timestamp"`
	Status       int64    `json:"status"`
}

type User struct {
	Id 				int 				`json:"id"`
	Login        	string   			`json:"login"`
	HashPassword 	[32]byte 			`json:"hash_password"`
	CreatedAt 		time.Time 			`json:"create_at"`

	Conn   			net.Conn
	Chats  			map[string][]Msg
	Online			bool
}

