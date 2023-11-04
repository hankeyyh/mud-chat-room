package main

import (
	"bytes"
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

type Client struct {
	conn net.Conn
	nick string
}

// ReadMessage 读取消息
func (c *Client) ReadMessage(buf []byte) (string, error) {
	bb := bytes.Buffer{}
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			return "", err
		}
		str := string(buf[:n])
		if str[len(str)-1] == '\n' {
			break
		}
		bb.WriteString(str)
	}
	return bb.String(), nil
}

// SendMessage 发送消息
func (c *Client) SendMessage(msg string) error {
	_, err := c.conn.Write([]byte("\r\n" + msg + "\r\n"))
	return err
}

// ChatRoom 聊天室
type ChatRoom struct {
	Port     int
	Users    map[net.Conn]*Client
	UserLock sync.RWMutex
	NumUser  int
}

func NewChatRoom(port int) *ChatRoom {
	return &ChatRoom{
		Port:    port,
		Users:   make(map[net.Conn]*Client),
		NumUser: 0,
	}
}

func (cr *ChatRoom) Start() {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cr.Port))
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		client := &Client{conn: conn, nick: fmt.Sprintf("User%d", conn.RemoteAddr().(*net.TCPAddr).Port)}
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
	delete(cr.Users, client.conn)
	cr.NumUser--
	cr.UserLock.Unlock()

	if err = client.conn.Close(); err != nil {
		log.Printf("close error: %v", err)
	}
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
	welcomeMsg := fmt.Sprintf("Welcome to chat room!, there are %d users online", cr.NumUser)
	err := client.SendMessage(welcomeMsg)
	if err != nil {
		log.Printf("SendMessage error: %v", err)
	}

	cr.UserLock.Lock()
	cr.Users[client.conn] = client
	cr.NumUser++
	cr.UserLock.Unlock()

	buf := make([]byte, 1024)
	for {
		msg, err := client.ReadMessage(buf)
		if err != nil {
			cr.RemoveClient(client)
			if err != io.EOF {
				log.Printf("read error: %v", err)
			}
			return
		}
		msg = strings.TrimSpace(msg)
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
