package server

import (
	"context"
	"encoding/json"
	"github.com/aluka-7/cache"
	"github.com/aluka-7/game-gateway/dto"
	"github.com/aluka-7/game-gateway/utils/logger"
	"github.com/aluka-7/utils"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

type WsServer struct {
	gnet.BuiltinEventEngine
	// 用户连接
	connections sync.Map

	gatewayCfg *dto.GatewayConfig
	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	// 分布式服务id
	serviceId string
	eng       gnet.Engine
	// 客户端消息
	inMsg chan *dto.CommonReq
	// tcp 服务地址
	tcpAddr string
	// tcp 服务
	ts *TcpServer
	// 缓存提供器
	ce cache.Provider
	// 访问限制器
	limiter *rate.Limiter
}

func NewWsServer(gatewayCfg *dto.GatewayConfig, ce cache.Provider, tcpAddr string) gnet.EventHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &WsServer{
		connections: sync.Map{},
		gatewayCfg:  gatewayCfg,
		ctx:         ctx,
		cancel:      cancel,
		serviceId:   utils.RandString(12),
		inMsg:       make(chan *dto.CommonReq, 1),
		tcpAddr:     tcpAddr,
		ce:          ce,
		limiter:     rate.NewLimiter(rate.Limit(2000), 10), // 初始令牌10个，每秒产生200个令牌，相当于每秒允许同时200个连接进来
	}
}

func (wss *WsServer) OnBoot(eng gnet.Engine) gnet.Action {
	wss.eng = eng
	logger.Log.Info("\033[0;32;40mGateway WS Server Is Listening...\033[0m")

	outMsg := make(chan *dto.CommonRes, 1)
	// 启动 游戏交互服务 -----------------------------Start
	wss.ts = NewTcpServer(wss.tcpAddr, wss.ce, wss.gatewayCfg.GameList, wss.inMsg, outMsg)
	go wss.ts.Run()
	// 启动 游戏交互服务 -----------------------------End

	go func() {
		for msg := range outMsg {
			payload, err := json.Marshal(msg)
			if err != nil {
				logger.Log.Errorf("WsServer OnBoot Marshal Error: %+v", err)
			}
			//p := base64.StdEncoding.EncodeToString(payload)
			if msg.UserId != 0 { // 发给指定用户
				if c, ok := wss.connections.Load(msg.UserId); ok {
					if conn, ok := c.(gnet.Conn); ok {
						wsc, ok := conn.Context().(*wsCodec)
						if !ok || wsc.String("server") != msg.Server {
							continue
						}
						err := wsutil.WriteServerBinary(conn, payload)
						if err != nil {
							logger.Log.Errorf("WsServer WriteServerMessage Error: %+v", err)
						}
					}
				}
			} else { // 发给对应游戏服务的所有用户
				wss.connections.Range(func(uid, c any) bool {
					if conn, ok := c.(gnet.Conn); ok {
						wsc, ok := conn.Context().(*wsCodec)
						if !ok || wsc.String("server") != msg.Server {
							return true
						}
						err := wsutil.WriteServerBinary(conn, payload)
						if err != nil {
							logger.Log.Errorf("WsServer WriteServerMessage Error: %+v", err)
						}
					}
					return true
				})
			}
		}
	}()
	return gnet.None
}

func (wss *WsServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	c.SetContext(new(wsCodec))
	if err := wss.limiter.Wait(wss.ctx); err != nil {
		logger.Log.Infof("WsServer OnOpen limiter Wait Error: %+v", err)
		return nil, gnet.Close
	}
	return nil, gnet.None
}

func (wss *WsServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	wsc, ok := c.Context().(*wsCodec)
	if !ok {
		return gnet.Close
	}
	if err != nil {
		logger.Log.Errorf("WsServer OnClose Error:%+v", err)
	}
	// 删除对应连接
	wss.connections.Delete(wsc.UID())

	logger.Log.Infof("user_id[%d] disconnected", wsc.UID())
	return gnet.None
}

func (wss *WsServer) OnTraffic(c gnet.Conn) (action gnet.Action) {
	wsc, ok := c.Context().(*wsCodec)
	if !ok {
		return gnet.Close
	}
	if wsc.readBufferBytes(c) == gnet.Close {
		return gnet.Close
	}
	ok, params, action := wsc.upgrade(c)
	if !ok {
		return
	}
	// 获取用户信息并进行会话绑定 --------------------------- Start
	if len(params) > 0 {
		if sid, ok := params["sid"]; ok {
			us := wsc.getSession(wss.ctx, wss.ce, sid)
			if us.Id == 0 {
				return gnet.Close
			}
			// 查找获取旧连接并关闭
			if oc, ok := wss.connections.Load(us.Id); ok {
				if conn, ok := oc.(gnet.Conn); ok {
					logger.Log.Infof("user_id[%d] disconnected", wsc.UID())
					conn.Close()
				}
				wss.connections.Delete(us.UserId)
			}
			wsc.Bind(us.Id)
			wss.connections.Store(us.Id, c)
		} else {
			return gnet.Close
		}
	}
	// 获取用户信息并进行会话绑定 --------------------------- End

	if wsc.buf.Len() <= 0 {
		return gnet.None
	}
	messages, err := wsc.Decode(c)
	if err != nil {
		return gnet.Close
	}
	if messages == nil {
		return
	}
	for _, message := range messages {
		// 自定义心跳 ------------------ Start
		if utils.Bytes2Str(message.Payload) == "Ping" {
			wsutil.WriteServerBinary(c, utils.Str2Bytes("Pong"))
			continue
		}
		// 自定义心跳 ------------------ Start

		var req = new(dto.CommonReq)
		err = json.Unmarshal(message.Payload, req)
		if err != nil {
			logger.Log.Errorf("user_id[%d] [err=%v]", wsc.UID(), err.Error())
			continue
		}
		if utils.Contains(wss.gatewayCfg.GameList, req.Server) > -1 {
			wsc.Set("server", req.Server)
			// 发送消息到游戏服务器
			req.UserId = wsc.UID()
			wss.inMsg <- req
		}
	}
	return gnet.None
}

func (wss *WsServer) OnTick() (delay time.Duration, action gnet.Action) {
	logger.Log.Infof("\033[0;33;40m[connected-count=%v]\033[0m", wss.eng.CountConnections())
	return 60 * time.Second, gnet.None
}

func (wss *WsServer) OnShutdown(eng gnet.Engine) {
	// 停止tcp服务
	wss.ts.Stop()
	logger.Log.Info("\033[0;33;40mGateway Ws Server Will Be Shutdown!\033[0m")
	wss.cancel()
}
