package conn

import "github.com/panjf2000/gnet/v2"

type Client struct {
	UID           int64
	Conn          gnet.Conn
	Send          chan []byte
	LastHeartbeat int64 // 最后心跳时间戳
}

func NewClient(uid int64, conn gnet.Conn) *Client {
	return &Client{
		UID:  uid,
		Conn: conn,
		Send: make(chan []byte, 64),
	}
}
