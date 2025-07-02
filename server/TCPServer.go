package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

type TCPServer struct {
	listener net.Listener
	domain   string
	port     int

	realConnection sync.WaitGroup

	logger     *log.Logger
	fileLogger *os.File
}

func NewTCPServer(domain string, port int) (*TCPServer, error) {
	listener, err := net.Listen("tcp", domain+":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	logger := log.Default()
	fileLogger, err := os.Create("logs/tcp_server.log")
	if err != nil {
		return nil, err
	}
	logger.SetOutput(fileLogger)
	logger.Println("Starting TCP Server")
	server := &TCPServer{listener: listener, domain: domain, port: port, logger: logger, fileLogger: fileLogger}
	return server, nil
}

func (server *TCPServer) Close() {
	server.logger.Println("Closing TCP Server")
	server.realConnection.Wait()
	server.listener.Close()
	server.fileLogger.Close()
}

func sendMessage(conn net.Conn, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(jsonData, '\n'))
	return err
}
