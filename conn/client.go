package conn

import "github.com/panjf2000/gnet/v2"

type Client struct {
	UID  int64
	Conn gnet.Conn
	Send chan []byte
}

func NewClient(uid int64, conn gnet.Conn) *Client {
	return &Client{
		UID:  uid,
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
