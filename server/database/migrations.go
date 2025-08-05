package database

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

func RunMigrations(DB *sql.DB) error {
	err := CreateUserTable(DB)
	if err != nil {
		return err
	}
	err = CreateConversationTable(DB)
	if err != nil {
		return err
	}
	err = CreateMsgsTable(DB)
	return err
}

func CreateUserTable(DB *sql.DB) error {
	query := `
        CREATE TABLE IF NOT EXISTS users (
            id INT PRIMARY KEY AUTO_INCREMENT,
            login VARCHAR(50) UNIQUE NOT NULL, 
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            password TEXT,
            online BOOLEAN DEFAULT FALSE
        );`
	_, err := DB.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func CreateConversationTable(DB *sql.DB) error {
	query := `
        CREATE TABLE IF NOT EXISTS conversations (
            id INT PRIMARY KEY AUTO_INCREMENT,
            user1_id INT NOT NULL,
            user2_id INT NOT NULL, 
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            UNIQUE KEY unique_pair (user1_id, user2_id), 
            FOREIGN KEY (user1_id) REFERENCES users(id) ON DELETE CASCADE,
            FOREIGN KEY (user2_id) REFERENCES users(id) ON DELETE CASCADE
        );`
	_, err := DB.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func CreateMsgsTable(DB *sql.DB) error {
	query := `
        CREATE TABLE IF NOT EXISTS messages (
            id INT PRIMARY KEY AUTO_INCREMENT,
            conversation_id INT NOT NULL,
            sender_id INT NOT NULL, 
            body TEXT NOT NULL,
            sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
            FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
        );`
	_, err := DB.Exec(query)
	if err != nil {
		return err
	}
	return nil
}
