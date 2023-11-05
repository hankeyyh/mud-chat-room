package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

var (
	serverPort = flag.Int("port", 1234, "server port")
)

// ChatRoom 聊天室
type ChatRoom struct {
	Port     int
	Users    map[int]*Client
	UserLock sync.RWMutex
	NumUser  int
	seq int
}

func NewChatRoom(port int) *ChatRoom {
	return &ChatRoom{
		Port:    port,
		Users:   make(map[int]*Client),
		NumUser: 0,
	}
}

func (cr *ChatRoom) Start() {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cr.Port))
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	fmt.Println("ChatRoom Start!")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		
		cr.seq += 1
		client := &Client{
			clientConn: NewClientConn(conn),
			id:         cr.seq,
			nick:       fmt.Sprintf("User%d", conn.RemoteAddr().(*net.TCPAddr).Port),
		}

		go cr.HandleClient(client)
	}
}

// RemoveClient 移除client
func (cr *ChatRoom) RemoveClient(client *Client) {
	err := client.SendMessage("Bye")
	if err != nil {
		log.Printf("write error: %v", err)
	}

	cr.UserLock.Lock()
	delete(cr.Users, client.id)
	cr.NumUser--
	cr.UserLock.Unlock()

	if err = client.Close(); err != nil {
		log.Printf("close error: %v", err)
	}
}

// AddClient 增加client
func (cr *ChatRoom) AddClient(client *Client) error {
	err := client.SendMessage(fmt.Sprintf("Welcome to chat room!, there are %d users online", cr.NumUser))
	if err != nil {
		log.Printf("SendMessage error: %v", err)
		return err
	}

	cr.UserLock.Lock()
	defer cr.UserLock.Unlock()
	cr.Users[client.id] = client
	cr.NumUser++

	return nil
}

// Broadcast 广播消息
func (cr *ChatRoom) Broadcast(msg string) {
	for _, client := range cr.Users {
		err := client.SendMessage(msg)
		if err != nil {
			log.Printf("write error: %v", err)
		}
	}
}

// HandleClient 持续接收用户消息并处理
func (cr *ChatRoom) HandleClient(client *Client) {
	// 加入用户
	if err := cr.AddClient(client); err != nil {
		return
	}

	for {
		msg, err := client.ReadMessage()
		msg = strings.TrimSpace(msg)
		if err != nil {
			cr.RemoveClient(client)
			if err != io.EOF {
				log.Printf("read error: %v", err)
			}
			return
		}
		if msg == "" {
			continue
		}

		if msg == "exit" {
			cr.RemoveClient(client)
			cr.Broadcast("User " + client.nick + " has left the chat room")
			return
		}

		// 改名
		if msg[0] == '/' {
			parts := strings.SplitN(msg, " ", 2)
			if parts[0] == "/nick" && len(parts) > 1 {
				newName := parts[1]
				client.nick = newName
				client.SendMessage("Your nickname has been changed to " + newName)
			}
			continue
		}

		msg = fmt.Sprintf("%s: %s", client.nick, msg)
		cr.Broadcast(msg)
	}
}

func main() {
	flag.Parse()

	chatRoom := NewChatRoom(*serverPort)
	chatRoom.Start()
}
