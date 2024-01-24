package server

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/aluka-7/cache"
	"github.com/aluka-7/game-gateway/dto"
	"github.com/aluka-7/game-gateway/utils/logger"
	"github.com/aluka-7/utils"
	"net"
	"strings"
	"sync"
)

type TcpServer struct {
	addr string

	gameConn sync.Map
	gameList []string

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	// 分布式服务id
	connected int64
	inMsg     chan *dto.CommonReq
	outMsg    chan *dto.CommonRes

	ce cache.Provider
}

func NewTcpServer(addr string, ce cache.Provider, gameList []string, inMsg chan *dto.CommonReq, outMsg chan *dto.CommonRes) *TcpServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &TcpServer{
		addr:     addr,
		ctx:      ctx,
		gameConn: sync.Map{},
		gameList: gameList,
		cancel:   cancel,
		inMsg:    inMsg,
		outMsg:   outMsg,
		ce:       ce,
	}
}

// Run ...
func (ts *TcpServer) Run() {
	listener, err := net.Listen("tcp", ts.addr)
	if err != nil {
		logger.Log.Errorf("TcpServer Run Error: %+v", err)
		return
	}
	defer listener.Close()
	logger.Log.Info("\033[0;32;40mGateway TCP Server Is Listening...\033[0m")

	go func() {
		for msg := range ts.inMsg {
			b, _ := json.Marshal(msg)
			if c, ok := ts.gameConn.Load(msg.Server); ok { // 发给对应游戏服务
				_, err := c.(net.Conn).Write(append(b, '\n'))
				if err != nil {
					logger.Log.Errorf("TcpServer Write Error: %+v", err)
				}
				logger.Log.Infof("TcpServer Write Msg To GameServer : %+v", string(b))
			}
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Log.Errorf("TcpServer Run Accept Error: %+v", err)
			return
		}
		reader := bufio.NewReader(conn)
		alias, err := reader.ReadString('\n') // 获取游戏服务别名
		if err != nil {
			logger.Log.Errorf("TcpServer Run ReadString Error: %+v", err)
			continue
		}
		if utils.Contains(ts.gameList, strings.TrimSpace(alias)) < 0 {
			continue
		}
		go ts.handleRequest(strings.TrimSpace(alias), conn)
	}
}

func (ts *TcpServer) Stop() {
	close(ts.inMsg)
	close(ts.outMsg)
}

func (ts *TcpServer) handleRequest(alias string, conn net.Conn) {
	defer conn.Close() // 关闭连接

	// 新增游戏
	ts.gameConn.Store(alias, conn)
	defer ts.gameConn.Delete(alias)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var buf = scanner.Bytes()
		logger.Log.Infof("Message incoming: %s", string(buf))
		var res = new(dto.CommonRes)
		err := json.Unmarshal(buf, res)
		if err != nil {
			logger.Log.Errorf("TcpServer handleRequest Unmarshal Error: %+v", err)
			break
		}
		res.Server = alias
		ts.outMsg <- res
	}
}
