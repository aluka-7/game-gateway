package dto

import (
	"encoding/json"
	"fmt"
	"github.com/aluka-7/utils"
)

type WsConfig struct {
	Addr string `json:"addr"`
}

type TcpConfig struct {
	Addr string `json:"addr"`
}

type Gateway struct {
	Path   string
	Config GatewayConfig
}

type GatewayConfig struct {
	GameList []string `json:"gameList"`
}

func (b *Gateway) Changed(data map[string]string) {
	if v, ok := data[b.Path]; ok {
		err := json.Unmarshal(utils.Str2Bytes(v), &b.Config)
		if err != nil {
			return
		}
	} else {
		panic(fmt.Sprintf("配置中心不存在[%s]配置", b.Path))
	}
}
