package tcp

import (
	"bufio"
	"net"
	"sync"
	"time"
)

const (
	defaultWriteTimeout = 3 * time.Second
	defaultReadBufSize  = 256 * 1024
	defaultSendBufSize  = 1024
)

type gameSession struct {
	alias  string
	conn   net.Conn
	reader *bufio.Reader

	send chan []byte
	once sync.Once
}

func newGameSession(alias string, conn net.Conn, reader *bufio.Reader) *gameSession {
	return &gameSession{
		alias:  alias,
		conn:   conn,
		reader: reader,
		send:   make(chan []byte, defaultSendBufSize),
	}
}

func (gs *gameSession) close() {
	gs.once.Do(func() {
		close(gs.send)
		_ = gs.conn.Close()
	})
}
