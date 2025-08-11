package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
	"strconv"
	"strings"
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

	conn  net.Conn
	chats map[string][]Msg

	fileLogger *os.File
	logger     *log.Logger

	mutex sync.Mutex
}

var (
	stop bool = false
)

func registerOrAuth(ip string, port int, scanner *bufio.Scanner, status int64) *User {
	conn, err := net.Dial("tcp", ip + ":" + strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}

	err = os.Mkdir("logs", 0666)
	if err != nil && err.(*os.PathError).Err.Error() != "file exists" {
		fmt.Println(err)
		return nil
	}
	f, err := os.Create("logs/client.log")
	if err != nil {
		fmt.Println(err)
		return nil
	}
	logger := log.Default()
	logger.SetOutput(f)
	fmt.Print("Write login: ")
	scanner.Scan()
	login := scanner.Text()
	for !checkLoginIsCorrect(login) {
		fmt.Print("Incorrect login. Write login: ")
		scanner.Scan()
		login = scanner.Text()
	}
	fmt.Print("Write password: ")
	scanner.Scan()
	password := scanner.Text()
	authMsg := AuthMsg{
		Login:        login,
		HashPassword: sha256.Sum256([]byte(password)),
		Status:       status,
		Timestamp:    time.Now().Unix(),
	}
	err = sendMessage(conn, authMsg)
	if err != nil {
		logger.Println("Error sending auth message:", err)
		fmt.Println("Error sending auth message:", err)
		return nil
	}

	reader := bufio.NewReader(conn)
	resp, err := reader.ReadString('\n')
	if err != nil {
		logger.Println("Error reading response:", err)
		return nil
	}
	err = json.Unmarshal([]byte(resp), &authMsg)
	if err != nil {
		logger.Println("Error unmarshalling response:", err)
		return nil
	}
	if authMsg.Status != 0 {
		fmt.Println("Incorrect password")
		return nil
	}
	user := User{
		Login:        login,
		HashPassword: sha256.Sum256([]byte(password)),

		conn:       conn,
		logger:     logger,
		fileLogger: f,
		chats:      make(map[string][]Msg),
	}
	return &user
}

func sendMessage(conn net.Conn, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(jsonData, '\n'))
	return err
}

func checkLoginIsCorrect(login string) bool {
	if strings.IndexFunc(login, func(c rune) bool {
		return !(('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9'))
	}) != -1 {
		return false
	}
	return true
}

func clearScreen() {
	fmt.Print("\033[2J") // Очищает экран
	fmt.Print("\033[H")  // Перемещает курсор в левый верхний угол
}

func handleUser(user *User) {

	//обрабатываем получаемые сообщения
	go func() {
		reader := bufio.NewReader(user.conn)
		var msg Msg
		for {
			input, err := reader.ReadString('\n')
			if err != nil {
				user.logger.Println("Error reading input:", err)
				return
			}
			if len(input) == 0 {
				continue
			}
			err = json.Unmarshal([]byte(input), &msg)
			if err != nil {
				user.logger.Println("Error unmarshalling input:", err)
			}
			fmt.Println(msg)
			user.mutex.Lock()
			if msg.Status != 0 {
				fmt.Println(msg.Text)
				user.logger.Println("Error: " + msg.Text)
			} else if msg.Sender == user.Login {
				user.chats[msg.Receiver] = append(user.chats[msg.Receiver], msg)
			} else {
				user.chats[msg.Sender] = append(user.chats[msg.Sender], msg)
			}
			user.mutex.Unlock()
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		clearScreen()
		if stop {
			break
		}
		fmt.Println("DIALOGS")
		user.mutex.Lock()
		for login, msgs := range user.chats {
			fmt.Println(login, msgs[len(msgs)-1].Sender, time.Unix(msgs[len(msgs)-1].Timestamp, 0).Format("2006-01-02 15:04:05"), msgs[len(msgs)-1].Text)
		}
		user.mutex.Unlock()
		fmt.Println("You can:\n1.Change dialog\n2.Exit")
		scanner.Scan()
		text := scanner.Text()
		if len(text) == 0 {
			continue
		}
		if text == "1" || text == "Change dialog" {
			fmt.Print("Write username: ")
			scanner.Scan()
			text = scanner.Text()
			handleDialog(user, text)
		} else if text == "2" || text == "Exit" {
			msg := Msg{
				Sender:    user.Login,
				Receiver:  user.Login,
				Timestamp: time.Now().Unix(),
				Text:      "disconnect",
				Status:    1,
			}
			err := sendMessage(user.conn, msg)
			if err != nil {
				user.logger.Println("Error sending disconnect message:", err)
				return
			}

			return
		}

	}
}

func handleDialog(user *User, dialog string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		clearScreen()
		if stop {
			break
		}
		user.mutex.Lock()
		for _, msg := range user.chats[dialog] {
			fmt.Println(msg.Sender, time.Unix(msg.Timestamp, 0).Format("2006-01-02 15:04:05"), msg.Text)
		}
		user.mutex.Unlock()
		scanner.Scan()
		text := scanner.Text()
		if len(text) == 0 {
			continue
		}
		if text == "exit" {
			break
		}
		msg := Msg{
			Sender:    user.Login,
			Receiver:  dialog,
			Timestamp: time.Now().Unix(),
			Text:      text,
		}
		err := sendMessage(user.conn, msg)
		if err != nil {
			user.logger.Println("Error sending disconnect message:", err)
			return
		}
	}
}

func (user *User) Close() {
	user.conn.Close()
	user.fileLogger.Close()
}

func main() {
	ip := "localhost"
	port := 14232
	var user *User
	defer func() {
		if user != nil {
			user.Close()
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if stop {
			break
		}
		fmt.Println("You can:\n1.Register\n2.Auth\n3.Exit")
		scanner.Scan()
		text := scanner.Text()
		if text == "1" || text == "Register" {
			user = registerOrAuth(ip, port, scanner, 0)
			if user == nil {
				continue
			}
			handleUser(user)
		} else if text == "2" || text == "Auth" {
			user = registerOrAuth(ip, port, scanner, 1)
			if user == nil {
				continue
			}
			handleUser(user)
		} else if text == "3" || text == "Exit" {
			break
		} else {
			fmt.Println("Invalid input")
		}
	}

}