package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/aluka-7/cache"
	"github.com/aluka-7/game-gateway/dto"
	"github.com/aluka-7/game-gateway/utils/logger"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

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
	fmt.Println(fmt.Sprintf("⇨ tcp server started on \u001B[0;32;40m%s\u001B[0m", ts.addr))

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
		packet, err := EncodeReq(msg)
		if err != nil {
			logger.Log.Errorf("TcpServer encode req error: %+v", err)
			continue
		}
		if c, ok := ts.gameConn.Load(msg.Server); ok { // 发给对应游戏服务
			session := c.(*gameSession)
			if !ts.enqueueMessage(session, packet) {
				logger.Log.Warnf("TcpServer drop msg to game server %s due to full queue", msg.Server)
				continue
			}
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
	if len(ts.allowedGames) == 0 {
		return true
	}
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

	for {
		payload, err := ReadFrame(session.reader)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			logger.Log.Errorf("TcpServer read frame error: %+v", err)
			return
		}

		res, err := DecodeRes(payload, alias)
		if err != nil {
			logger.Log.Errorf("TcpServer decode protobuf response error: %+v", err)
			continue
		}
		select {
		case ts.outMsg <- res:
		case <-ts.ctx.Done():
			return
		}
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
