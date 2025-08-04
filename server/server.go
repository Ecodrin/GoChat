package main

import (
	"bufio"
	"encoding/json"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"


	"server/utility"
	"server/handlers"
	"server/database"


	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)


type Server struct {

	tcpServer 				*TCPServer
	mutex     				sync.Mutex

	logger     				*log.Logger
	loggerFile 				*os.File

	kafkaProducer         	*kafka.Producer
	kafkaMsgTopic         	string
	kafkaBootstrapServers 	string

	DB 						*sql.DB
	Conns 					map[int]net.Conn
}

func NewServer(domain string, port int, kafkaBootstrapServers string, kafkaMsgTopic string, dataSourceName string) (*Server, error) {
	logger := log.Default()
	err := os.Mkdir("logs", 0666)
	if err != nil && err.(*os.PathError).Err.Error() != "file exists"{
		return nil, err
	}
	f, err := os.Create("logs/server.log")
	if err != nil {
		return nil, err
	}

	tcpServer, err := NewTCPServer(domain, port)
	if err != nil {
		f.Close()
		return nil, err
	}

	logger.SetOutput(f)

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaBootstrapServers,
	})
	if err != nil {
		f.Close()
		tcpServer.Close()
		return nil, err
	}

	DB, err := database.InitDb(dataSourceName)
	if err != nil {
		logger.Println("error in init db: " + err.Error())
		f.Close()
		tcpServer.Close()
		producer.Close()
		return nil, err
	}
	logger.Println("database init successful")

	server := &Server{
		logger:                	logger,
		loggerFile:            	f,
		tcpServer:             	tcpServer,
		kafkaProducer:         	producer,
		kafkaMsgTopic:         	kafkaMsgTopic,
		kafkaBootstrapServers: 	kafkaBootstrapServers,
		DB:						DB,
		Conns:					make(map[int]net.Conn),
	}

	err = database.RunMigrations(server.DB)
	if err != nil {
		logger.Println("error in runMigrationsL: " + err.Error())
		return nil, err
	}
	return server, nil
}

func (server *Server) ConsumerWork() {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": server.kafkaBootstrapServers,
		"group.id":          "myGroup",
		"auto.offset.reset": "latest",
	})
	if err != nil {
		server.logger.Fatal(err)
		return
	}
	server.logger.Println("Kafka consumer init")
	defer consumer.Close()

	if err := consumer.SubscribeTopics([]string{server.kafkaMsgTopic}, nil); err != nil {
		server.logger.Fatal(err)
		consumer.Close()
		return
	}

	for {
		msg, err := consumer.ReadMessage(time.Second)
		if err != nil {
			//server.logger.Println("consumer: " + err.Error())
			continue
		}

		var msgJSON handlers.Msg
		err = json.Unmarshal(msg.Value, &msgJSON)
		if err != nil {
			server.logger.Println(err.Error())
			continue
		}
		server.mutex.Lock()

		
		userReceiver, err := database.GetUserByLogin(server.DB, msgJSON.Receiver)

		if err == nil {
			userSender, err := database.GetUserByLogin(server.DB, msgJSON.Sender)
			if err != nil {
				server.logger.Println("user " + msgJSON.Sender + " not found")
			} else {
				err = database.AddMessageToConversation(server.DB, userReceiver.Id, userSender.Id, msgJSON.Text, time.Unix(msgJSON.Timestamp, 0))
				if err != nil {
					server.logger.Println(err)
				} else {
					if userSender.Online {
						err = sendMessage(server.Conns[userSender.Id], msgJSON)
						if err != nil {
							server.logger.Println("kafka " + userSender.Login + " " + err.Error())
						}
					}
					if userReceiver.Online {
						err = sendMessage(server.Conns[userReceiver.Id], msgJSON)
						if err != nil {
							server.logger.Println("kafka " + userReceiver.Login + " " + err.Error())
						}
					}
					server.logger.Println(userReceiver.Login + " send to " + userSender.Login + " msg")
				}
			}
		} else {
			server.logger.Println("user " + msgJSON.Receiver + " not found")
			errorMsg := handlers.Msg{
				Sender: msgJSON.Sender,
				Receiver: msgJSON.Receiver,
				Status: 1,
				Text: "incorrect user",
			}
			userSender, err := database.GetUserByLogin(server.DB, msgJSON.Sender)
			if err != nil {
				server.logger.Println("user " + msgJSON.Receiver + " not found")
			} else {
				if userSender.Online {
					err = sendMessage(server.Conns[userSender.Id], errorMsg)
					if err != nil {
						server.logger.Println("kafka " + userSender.Login + " " + err.Error())
					}
				}
			}
		}
		server.mutex.Unlock()
	}
}

func (server *Server) start() {

	server.logger.Println("Server start")
	go server.ConsumerWork()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			conn, err := server.tcpServer.listener.Accept()
			if err != nil {
				server.logger.Println(err.Error())
				continue
			}
			go server.handleConnection(conn)
		}

	}()

	<-sigchan
	server.logger.Println("Shutting down server...")
	server.Close()
}

func (server *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	var msg handlers.AuthMsg

	data, err := reader.ReadString('\n')
	if err != nil {
		server.logger.Println(err.Error())
		return
	}
	err = json.Unmarshal([]byte(data), &msg)
	if err != nil {
		server.logger.Println(err.Error())
		return
	}


	server.mutex.Lock()
	user, err := database.GetUserByLogin(server.DB, msg.Login)
	server.mutex.Unlock()

	if (err == sql.ErrNoRows && msg.Status == 1) || (err == nil && user.HashPassword != utility.ToHex(msg.HashPassword)) || (err == nil && msg.Status == 0) || (err == nil && user.Online) {
		server.logger.Println("user " + msg.Login + " not found or invalid password")
		output := handlers.AuthMsg{Login: msg.Login, HashPassword: msg.HashPassword, Status: 2, }
		err = sendMessage(conn, output)
		return
	}

	server.mutex.Lock()
	fl := true
	if err == sql.ErrNoRows {
		user = &handlers.User {
			Login: 			msg.Login,
			HashPassword: 	utility.ToHex(msg.HashPassword),
			CreatedAt: 		time.Unix(msg.Timestamp, 0),
			Online: 		true,
		}
		fl = false
		err = database.CreateUser(server.DB, user)
		if err != nil {
			server.logger.Println(err)
		}

		user, err = database.GetUserByLogin(server.DB, user.Login)
		if err != nil {
			server.logger.Println(err)
			server.mutex.Unlock()
			return
		}
	}

	server.Conns[user.Id] = conn
	err = database.UpdateUserOnline(server.DB, user.Id, true)
	if err != nil {
		server.logger.Println(err)
		server.mutex.Unlock()
		return
	}

	server.logger.Println("User " + msg.Login + " online at " + time.Now().Format("2006-01-02 15:04:05"))

	server.mutex.Unlock()

	err = sendMessage(conn, handlers.AuthMsg{Login: msg.Login, HashPassword: msg.HashPassword, Status: 0})
	if err != nil {
		server.logger.Println(err.Error())
		return
	}

	// если пользователь уже сущестовал, отправляем ему все предыдущие сообщения
	if fl {
		server.mutex.Lock()

		msgs, err := database.GetMessagesBySenderID(server.DB, user.Id)
		if err != nil {
			server.logger.Println(err.Error())
			server.mutex.Unlock()
			return
		}
		for _, imsg := range msgs {
				user1Login, user2Login, err := database.GetUsersLoginsByConversaionId(server.DB, imsg.ConversationId)
				if err != nil {
					server.logger.Println("get old msgs: " + err.Error())
				}
				outMsg := handlers.Msg {
					Timestamp: imsg.SentAt.Unix(),
					Text: imsg.Body,
					Status: 0,
				}
				if user1Login == user.Login {
					outMsg.Sender = user1Login
					outMsg.Receiver = user2Login
				} else {
					outMsg.Sender = user2Login
					outMsg.Receiver = user1Login
				}

				err = sendMessage(conn, outMsg)
				if err != nil {
					server.logger.Println("get old msgs: " + err.Error())
				}
			}
		server.mutex.Unlock()
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			server.logger.Println(err.Error())
			return
		}

		var msg handlers.Msg
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			server.logger.Println("SS00 " + err.Error())
			continue
		}
		if msg.Status == 1 {
			server.mutex.Lock()
			delete(server.Conns, user.Id)
			err = database.UpdateUserOnline(server.DB, user.Id, false)
			if err != nil {
				server.logger.Println(err.Error())
			} else {
				server.logger.Println("User " + user.Login + " disconnected")
			}
			server.mutex.Unlock()
			break
		}

		err = server.sendToKafka(msg)
		if err != nil {
			server.logger.Println(err.Error())
		}
	}

}

func (server *Server) Close() {
	server.logger.Println("Closing server...")
	server.tcpServer.Close()
	server.loggerFile.Close()
	server.kafkaProducer.Close()
	server.DB.Close()
}

func (server *Server) sendToKafka(msg interface{}) error {
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		server.logger.Println(err.Error())
		return err
	}

	return server.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &server.kafkaMsgTopic,
			Partition: kafka.PartitionAny,
		},
		Value: jsonMsg,
	}, nil)
}

func main() {

	server, err := NewServer("localhost", 14232, ":9092", "msgTopic", "root:" + SecurityMySQLRootPassword + "@tcp(localhost:3306)/f.db?parseTime=true")
	if err != nil {
		server.logger.Fatal(err)
		panic(err)
	}
	server.start()
	server.Close()
}