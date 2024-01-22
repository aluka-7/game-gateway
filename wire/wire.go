//go:build wireinject
// +build wireinject

//go:generate wire

package wire

import (
	"github.com/aluka-7/cache"
	"github.com/aluka-7/game-gateway/dto"
	"github.com/aluka-7/game-gateway/server"
	"github.com/google/wire"
	"github.com/panjf2000/gnet/v2"
)

const (
	SystemId = "10000"
)

func InitializeWsServer(*dto.GatewayConfig, cache.Provider, string) gnet.EventHandler {
	panic(wire.Build(server.NewWsServer))
}
