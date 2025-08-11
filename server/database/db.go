package database

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"server/handlers"
	"time"
)

func InitDb(dataSourceName string) (*sql.DB, error) {
	DB, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}
	err = DB.Ping()
	if err != nil {
		return nil, err
	}
	return DB, nil
}

func CreateUser(DB *sql.DB, user *handlers.User) error {
	_, err := DB.Exec("INSERT INTO users (login, created_at, password) VALUES(?, ?, ?)", user.Login, user.CreatedAt, user.HashPassword)
	return err
}

func GetUserById(DB *sql.DB, id int) (*handlers.User, error) {
	var user handlers.User
	err := DB.QueryRow("SELECT id, login, created_at, password FROM users WHERE id = ?", id).Scan(
		&user.Id, &user.Login, &user.CreatedAt, &user.HashPassword)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByLogin(DB *sql.DB, login string) (*handlers.User, error) {
	query := "SELECT id, login, password, created_at, online FROM users WHERE login = ?"
	row := DB.QueryRow(query, login)

	var user handlers.User
	err := row.Scan(&user.Id, &user.Login, &user.HashPassword, &user.CreatedAt, &user.Online)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func CreateConversation(DB *sql.DB, user1Id int, user2Id int) (*handlers.Conversation, error) {
	existingConv, err := GetConversationBetweenUsers(DB, user1Id, user2Id)
	if err == nil && existingConv != nil {
		return existingConv, nil
	}
	query := "INSERT INTO conversations (user1_id, user2_id) VALUES (?, ?)"
	result, err := DB.Exec(query, user1Id, user2Id)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	conversation := &handlers.Conversation{
		ID:        int(id),
		User1Id:   user1Id,
		User2Id:   user2Id,
		CreatedAt: time.Now(),
	}
	return conversation, nil
}

func GetConversationBetweenUsers(db *sql.DB, user1ID, user2ID int) (*handlers.Conversation, error) {
	query := `SELECT id, user1_id, user2_id, created_at FROM conversations 
				WHERE (user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)`
	row := db.QueryRow(query, user1ID, user2ID, user2ID, user1ID)

	var conversation handlers.Conversation
	err := row.Scan(&conversation.ID, &conversation.User1Id, &conversation.User2Id, &conversation.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

func GetConversationByID(db *sql.DB, id int) (*handlers.Conversation, error) {
	query := "SELECT id, user1_id, user2_id, created_at FROM conversations WHERE id = ?"
	row := db.QueryRow(query, id)

	var conversation handlers.Conversation
	err := row.Scan(&conversation.ID, &conversation.User1Id, &conversation.User2Id, &conversation.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, fmt.Errorf("error getting conversation: %v", err)
	}

	return &conversation, nil
}

func CreateMsg(DB *sql.DB, conversationID int, senderID int, body string, sentAt time.Time) (*handlers.DataBaseMsg, error) {
	query := "INSERT INTO messages (conversation_id, sender_id, body, sent_at) VALUES (?, ?, ?, ?)"
	result, err := DB.Exec(query, conversationID, senderID, body, sentAt)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	msg := handlers.DataBaseMsg{
		ID:             int(id),
		ConversationId: conversationID,
		SenderId:       senderID,
		Body:           body,
		SentAt:         sentAt,
	}
	return &msg, nil
}

func GetMsgById(DB *sql.DB, id int) (*handlers.DataBaseMsg, error) {
	query := "SELECT id, conversation_id, sender_id, body, sent_at FROM messages WHERE id = ?"
	row := DB.QueryRow(query, id)
	var msg handlers.DataBaseMsg
	err := row.Scan(&msg.ID, &msg.ConversationId, &msg.SenderId, &msg.Body, &msg.SentAt)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func GetMsgsByConversationID(DB *sql.DB, conversationID int) ([]handlers.DataBaseMsg, error) {
	query := "SELECT id, conversation_id, sender_id, body, sent_at FROM messages WHERE conversation_id = ? ORDER BY sent_at ASC"
	rows, err := DB.Query(query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []handlers.DataBaseMsg
	for rows.Next() {
		var msg handlers.DataBaseMsg
		err := rows.Scan(&msg.ID, &msg.ConversationId, &msg.SenderId, &msg.Body, &msg.SentAt)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func GetAllUserMessages(db *sql.DB, userID int) ([]handlers.DataBaseMsg, error) {
    query := `
        SELECT id, conversation_id, sender_id, body, sent_at 
        FROM messages 
        WHERE sender_id = ? OR conversation_id IN (
            SELECT id FROM conversations 
            WHERE user1_id = ? OR user2_id = ?
        )
        ORDER BY sent_at ASC`
    
    rows, err := db.Query(query, userID, userID, userID)
    if err != nil {
        return nil, fmt.Errorf("error getting messages: %v", err)
    }
    defer rows.Close()

    var messages []handlers.DataBaseMsg
    for rows.Next() {
        var msg handlers.DataBaseMsg
        err := rows.Scan(&msg.ID, &msg.ConversationId, &msg.SenderId, &msg.Body, &msg.SentAt)
        if err != nil {
            return nil, err
        }
        messages = append(messages, msg)
    }

    return messages, nil
}

func GetUsersByConversaionId(db *sql.DB, Id int) (*handlers.User, *handlers.User, error) {
	conv, err := GetConversationByID(db, Id)
	if err != nil {
		return nil, nil, err
	}

	user1, err := GetUserById(db, conv.User1Id)
	if err != nil {
		return nil, nil, err
	}
	user2, err := GetUserById(db, conv.User2Id)
	if err != nil {
		return nil, nil, err
	}

	return user1, user2, nil
}

func UpdateUserOnline(db *sql.DB, id int, online bool) error {
	onlineValue := 0
	if online {
		onlineValue = 1
	}

	query := "UPDATE users SET online = ? WHERE id = ?"
	result, err := db.Exec(query, onlineValue, id)
	if err != nil {
		return fmt.Errorf("error updating user online: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %d not found", id)
	}

	return nil
}

func AddMessageToConversation(DB *sql.DB, senderID, receiverID int, body string, sentAt time.Time) error {
	conversation, err := GetConversationBetweenUsers(DB, senderID, receiverID)
	if err != nil {
		conversation, err = CreateConversation(DB, senderID, receiverID)
		if err != nil {
			return err
		}
	}

	_, err = CreateMsg(DB, conversation.ID, senderID, body, sentAt)
	if err != nil {
		return err
	}
	return nil
}
