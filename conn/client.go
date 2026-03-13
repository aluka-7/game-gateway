package conn

import "github.com/panjf2000/gnet/v2"

type Client struct {
	UID  int64
	Conn gnet.Conn
	Send chan []byte
}

func NewClient(conn gnet.Conn) *Client {
	return &Client{
		Conn: conn,
		Send: make(chan []byte, 64),
	}
}

func (c *Client) Write(msg []byte) {
	select {
	case c.Send <- msg:

	default:
		c.Conn.Close()
	}
}
