package main

import (
	"fmt"
	"github.com/aluka-7/cache"
	_ "github.com/aluka-7/cache-redis"
	"github.com/aluka-7/configuration"
	"github.com/aluka-7/game-gateway/dto"
	"github.com/aluka-7/game-gateway/wire"
	"github.com/aluka-7/web"
	"github.com/labstack/echo/v4"
	"github.com/panjf2000/gnet/v2"
	"time"
)

func App(conf configuration.Configuration) {
	var wc dto.WsConfig
	if err := conf.Clazz("base", "server", "ws", wire.SystemId, &wc); err != nil {
		panic("加载WS运行配置出错")
	}
	var tc dto.TcpConfig
	if err := conf.Clazz("base", "server", "tcp", wire.SystemId, &tc); err != nil {
		panic("加载TCP运行配置出错")
	}

	ce := cache.Engine(wire.SystemId, conf)

	// 网关配置
	var gateway = &dto.Gateway{Path: fmt.Sprintf("/system/app/game/gateway")}
	conf.Get("app", "game", "", []string{"gateway"}, gateway)

	wss := wire.InitializeWsServer(&gateway.Config, ce, tc.Addr)

	web.App(func(eng *echo.Echo) {
		// Start serving!
		go func() {
			err := gnet.Run(wss, wc.Addr, gnet.WithMulticore(true), gnet.WithReusePort(true), gnet.WithTicker(true), gnet.WithTCPKeepAlive(time.Minute*5))
			if err != nil {
				panic(fmt.Sprintf("gnet run error: %+v", err))
			}
		}()
	}, wire.SystemId, conf)
}

func main() {
	conf := configuration.DefaultEngine()
	App(conf)
}
