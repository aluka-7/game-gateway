package tcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"github.com/aluka-7/cache"
	"github.com/aluka-7/game-gateway/dto"
	"github.com/aluka-7/game-gateway/utils/logger"
	"net"
	"strings"
	"sync"
	"sync/atomic"
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

type TcpServer struct {
	addr string

	listener net.Listener
	stopOnce sync.Once

	gameConn     sync.Map
	allowedGames map[string]struct{}

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	// 分布式服务id
	connected int64
	inMsg     <-chan *dto.CommonReq
	outMsg    chan<- *dto.CommonRes

	ce     cache.Provider
	closed atomic.Bool
}

func NewTcpServer(addr string, ce cache.Provider, gameList []string, inMsg <-chan *dto.CommonReq, outMsg chan<- *dto.CommonRes) *TcpServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &TcpServer{
		addr:         addr,
		ctx:          ctx,
		gameConn:     sync.Map{},
		allowedGames: buildAllowedGames(gameList),
		cancel:       cancel,
		inMsg:        inMsg,
		outMsg:       outMsg,
		ce:           ce,
	}
}

// Run ...
func (ts *TcpServer) Run() {
	listener, err := net.Listen("tcp", ts.addr)
	if err != nil {
		logger.Log.Errorf("TcpServer Run Error: %+v", err)
		return
	}
	ts.listener = listener
	defer listener.Close()
	logger.Log.Info("\033[0;32;40mGateway TCP Server Is Listening...\033[0m")

	go ts.dispatchLoop()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ts.closed.Load() || errors.Is(err, net.ErrClosed) {
				return
			}
			logger.Log.Errorf("TcpServer Run Accept Error: %+v", err)
			return
		}
		reader := bufio.NewReader(conn)
		alias, err := reader.ReadString('\n') // 获取游戏服务别名
		if err != nil {
			logger.Log.Errorf("TcpServer Run ReadString Error: %+v", err)
			_ = conn.Close()
			continue
		}
		cleanAlias := strings.TrimSpace(alias)
		if !ts.isAllowedGame(cleanAlias) {
			logger.Log.Warnf("TcpServer reject unknown game alias: %s", cleanAlias)
			_ = conn.Close()
			continue
		}
		go ts.handleRequest(cleanAlias, conn, reader)
	}
}

func (ts *TcpServer) dispatchLoop() {
	for msg := range ts.inMsg {
		b, err := json.Marshal(msg)
		if err != nil {
			logger.Log.Errorf("TcpServer dispatch marshal Error: %+v", err)
			continue
		}
		if c, ok := ts.gameConn.Load(msg.Server); ok { // 发给对应游戏服务
			session := c.(*gameSession)
			if !ts.enqueueMessage(session, append(b, '\n')) {
				logger.Log.Warnf("TcpServer drop msg to game server %s due to full queue", msg.Server)
				continue
			}
			logger.Log.Infof("TcpServer Write Msg To GameServer : %+v", string(b))
		}
	}
}

func (ts *TcpServer) enqueueMessage(session *gameSession, msg []byte) bool {
	select {
	case session.send <- msg:
		return true
	default:
		return false
	}
}

func buildAllowedGames(gameList []string) map[string]struct{} {
	allowedGames := make(map[string]struct{}, len(gameList))
	for _, game := range gameList {
		trimmed := strings.TrimSpace(game)
		if trimmed == "" {
			continue
		}
		allowedGames[trimmed] = struct{}{}
	}
	return allowedGames
}

func (ts *TcpServer) isAllowedGame(alias string) bool {
	_, ok := ts.allowedGames[alias]
	return ok
}

func (ts *TcpServer) Stop() {
	ts.stopOnce.Do(func() {
		ts.closed.Store(true)
		ts.cancel()
		if ts.listener != nil {
			_ = ts.listener.Close()
		}
		ts.gameConn.Range(func(_, value any) bool {
			value.(*gameSession).close()
			return true
		})
	})
}

func (ts *TcpServer) handleRequest(alias string, conn net.Conn, reader *bufio.Reader) {
	session := newGameSession(alias, conn, reader)
	if old, loaded := ts.gameConn.LoadOrStore(alias, session); loaded {
		oldSession := old.(*gameSession)
		oldSession.close()
		ts.gameConn.Store(alias, session)
	}
	defer func() {
		ts.gameConn.Delete(alias)
		session.close()
	}()

	go ts.writeToGameServer(session)

	scanner := bufio.NewScanner(session.reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, defaultReadBufSize)
	for scanner.Scan() {
		var payload = scanner.Bytes()
		logger.Log.Infof("Message incoming: %s", string(payload))
		var res = new(dto.CommonRes)
		err := json.Unmarshal(payload, res)
		if err != nil {
			logger.Log.Errorf("TcpServer handleRequest Unmarshal Error: %+v", err)
			break
		}
		res.Server = alias
		select {
		case ts.outMsg <- res:
		case <-ts.ctx.Done():
			return
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Log.Errorf("TcpServer scanner Error: %+v", err)
	}
}

func (ts *TcpServer) writeToGameServer(session *gameSession) {
	for msg := range session.send {
		_ = session.conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
		if _, err := session.conn.Write(msg); err != nil {
			logger.Log.Errorf("TcpServer Write Error: %+v", err)
			session.close()
			return
		}
	}
}
