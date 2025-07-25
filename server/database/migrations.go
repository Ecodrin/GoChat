package database


func RunMigrations(DB * sql.DB) {
	CreateUserTable(DB)
	CreateConversationTable(DB)
	CreateMsgsTable(DB)
}

func CreateUserTable(DB * sql.DB) {
	query := "
		CREATE TABLE users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(50) UNIQUE NOT NULL, 
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			password INT
	);"
	_, err := DB.Exec(query)
	if err != nil {
		fmt.Printlm("Error in CreateUserTable: " + err)
		return
	}
}

func CreateConversationTable(DB * sql.DB) {
	query := "
		CREATE TABLE conversations (
    		id INT PRIMARY KEY AUTO_INCREMENT,
    		user1_id INT NOT NULL,
    		user2_id INT NOT NULL, 
    		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    		UNIQUE KEY unique_pair (user1_id, user2_id), 
    		FOREIGN KEY (user1_id) REFERENCES users(id) ON DELETE CASCADE,
    		FOREIGN KEY (user2_id) REFERENCES users(id) ON DELETE CASCADE
	);"
	_, err := DB.Exec(query)
	if err != nil {
		fmt.Printlm("Error in CreateUserTable: " + err)
		return
	}
}


func CreateMsgsTable(DB * sql.DB) {
	query := "
		CREATE TABLE messages (
    		id INT PRIMARY KEY AUTO_INCREMENT,
    		conversation_id INT NOT NULL,
    		sender_id INT NOT NULL, 
    		body TEXT NOT NULL,
    		sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    		FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
);"
}