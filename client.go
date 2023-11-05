package main

import (
	"net"
	"strings"
)

type ClientConn struct {
	conn net.Conn
	buf  []byte
}

func NewClientConn(conn net.Conn) *ClientConn {
	return &ClientConn{
		conn: conn,
		buf:  make([]byte, 4),
	}
}

func (cc *ClientConn) ReadMessage(delim byte) (string, error) {
	builder := strings.Builder{}
	for {
		n, err := cc.conn.Read(cc.buf)
		if err != nil {
			return "", err
		}
		builder.Write(cc.buf[:n])
		if n == 0 || cc.buf[n-1] == delim {
			break
		}
	}
	return builder.String(), nil
}

func (cc *ClientConn) SendMessage(msg string) error {
	_, err := cc.conn.Write([]byte("\r\n" + msg + "\r\n"))
	return err
}

func (cc *ClientConn) Close() error {
	return cc.conn.Close()
}

// 用户
type Client struct {
	clientConn *ClientConn
	id         int
	nick       string
}

// ReadMessage 读取消息
func (c *Client) ReadMessage() (string, error) {
	return c.clientConn.ReadMessage('\n')
}

// SendMessage 发送消息
func (c *Client) SendMessage(msg string) error {
	return c.clientConn.SendMessage(msg)
}

// 关闭连接
func (c *Client) Close() error {
	return c.clientConn.Close()
}