package ws

import (
	"context"
	"encoding/json"
	"github.com/aluka-7/cache"
	"github.com/aluka-7/game-gateway/conn"
	"github.com/aluka-7/game-gateway/dto"
	"github.com/aluka-7/game-gateway/tcp"
	"github.com/aluka-7/game-gateway/utils/logger"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
	"golang.org/x/time/rate"
	"time"
)

type Server struct {
	gnet.BuiltinEventEngine

	engine gnet.Engine

	// 网关配置
	cfg *dto.GatewayConfig

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	// 连接管理器
	connMgr *conn.Manager

	// req 消息
	inMsg chan *dto.CommonReq
	// res 消息
	outMsg chan *dto.CommonRes

	// tcp 服务地址
	tcpAddr string
	// tcp 服务
	tcpSrv *tcp.TcpServer

	// 缓存提供器
	cache cache.Provider
	// 访问限制器
	limiter *rate.Limiter
}

func NewWsServer(cfg *dto.GatewayConfig, ce cache.Provider, tcpAddr string) gnet.EventHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		ctx:     ctx,
		cancel:  cancel,
		cfg:     cfg,
		cache:   ce,
		tcpAddr: tcpAddr,

		connMgr: conn.NewManager(),

		inMsg:   make(chan *dto.CommonReq, 1024),
		outMsg:  make(chan *dto.CommonRes, 1024),
		limiter: rate.NewLimiter(rate.Limit(2000), 10), // 初始令牌10个，每秒产生200个令牌，相当于每秒允许同时200个连接进来
	}
}

func (w *Server) OnBoot(eng gnet.Engine) gnet.Action {
	w.engine = eng
	logger.Log.Info("\033[0;32;40mGateway WS Server Started\033[0m")

	w.tcpSrv = tcp.NewTcpServer(
		w.tcpAddr,
		w.cache,
		w.cfg.GameList,
		w.inMsg,
		w.outMsg,
	)

	go w.tcpSrv.Run()
	go w.writeLoop()

	return gnet.None
}

// writeLoop 消息发送循环
func (w *Server) writeLoop() {
	for {
		select {
		case msg := <-w.outMsg:
			w.dispatch(msg)
		case <-w.ctx.Done():
			return
		}
	}
}

// dispatch 消息分发
func (w *Server) dispatch(msg *dto.CommonRes) {
	payload, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Error(err)
		return
	}
	if msg.UserId != 0 {
		w.sendToUser(msg.UserId, payload)
	} else {
		w.broadcast(msg.Server, payload)
	}
}

// sendToUser 发送给用户
func (w *Server) sendToUser(uid int64, payload []byte) {
	client, ok := w.connMgr.Get(uid)
	if !ok {
		return
	}
	err := wsutil.WriteServerBinary(client.Conn, payload)
	if err != nil {
		logger.Log.Error(err)
	}
}

// broadcast 广播
func (w *Server) broadcast(server string, payload []byte) {
	for _, item := range w.connMgr.Snapshot() {
		client := item.Client
		wsc := client.Conn.Context().(*wsCodec)
		if wsc.String("server") != server {
			continue
		}
		if err := wsutil.WriteServerBinary(client.Conn, payload); err != nil {
			logger.Log.Error(err)
		}
	}
}

func (w *Server) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	if !w.limiter.Allow() {
		return nil, gnet.Close
	}
	c.SetContext(new(wsCodec))
	return nil, gnet.None
}

func (w *Server) OnClose(c gnet.Conn, err error) gnet.Action {
	wsc := c.Context().(*wsCodec)
	uid := wsc.UID()
	if uid != 0 {
		w.connMgr.Remove(uid)
	}
	return gnet.None
}

func (w *Server) OnTraffic(c gnet.Conn) gnet.Action {
	wsc := c.Context().(*wsCodec)
	if wsc.readBufferBytes(c) == gnet.Close {
		return gnet.Close
	}
	ok, params, action := wsc.upgrade(c)
	if !ok {
		return action
	}
	if len(params) > 0 {
		if !w.bindUser(c, params) {
			return gnet.Close
		}
	}
	return w.handleMessages(c, wsc)
}

func (w *Server) bindUser(c gnet.Conn, params map[string]string) bool {
	sid := params["sid"]
	wsc := c.Context().(*wsCodec)
	us := wsc.getSession(w.ctx, w.cache, sid)
	if us.Id == 0 {
		return false
	}
	client := conn.NewClient(us.Id, c)
	w.connMgr.Add(us.Id, client)
	wsc.Bind(us.Id)
	return true
}

func (w *Server) handleMessages(c gnet.Conn, wsc *wsCodec) gnet.Action {
	messages, err := wsc.Decode(c)
	if err != nil {
		return gnet.Close
	}
	for _, msg := range messages {
		if string(msg.Payload) == "Ping" {
			wsutil.WriteServerBinary(c, []byte("Pong"))
			continue
		}
		var req dto.CommonReq
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			continue
		}
		req.UserId = wsc.UID()
		w.inMsg <- &req
	}
	return gnet.None
}

func (w *Server) OnTick() (delay time.Duration, action gnet.Action) {
	logger.Log.Infof("\033[0;33;40m[connected-count=%v]\033[0m", w.engine.CountConnections())
	return 60 * time.Second, gnet.None
}

func (w *Server) OnShutdown(eng gnet.Engine) {
	// 停止tcp服务
	close(w.inMsg)
	close(w.outMsg)
	w.tcpSrv.Stop()
	logger.Log.Info("\033[0;33;40mGateway Ws Server Will Be Shutdown!\033[0m")
	w.cancel()
}
