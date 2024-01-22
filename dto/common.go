package dto

type CommonReq struct {
	Server string       `json:"server"` // 服务
	UserId int64        `json:"userId"` // 用户id
	Msg    CommonReqMsg `json:"msg"`    // 消息
}

type CommonReqMsg struct {
	Event string      `json:"event"` // 事件
	Data  interface{} `json:"data"`  // 数据
}

// CommonRes 给客户端的消息
type CommonRes struct {
	UserId int64       `json:"userId,omitempty"` // 用户id，为 0 发给所有人
	Server string      `json:"server"`           // 服务名称
	Event  string      `json:"event"`            // 客户端事件
	Code   int         `json:"code"`             // 错误码
	Data   interface{} `json:"data"`             // 数据
}

// UserSession 用户会话结构体
type UserSession struct {
	Id            int64  `json:"id"`            // 用户id
	UserId        string `json:"userId"`        // 运营商用户id
	UserName      string `json:"userName"`      // 用户名
	AgentId       int64  `json:"agentId"`       // 所属代理id
	AgentName     string `json:"agentName"`     // 所属代理名称
	UserSessionId string `json:"userSessionId"` // 运营商用户 SessionId
	GameId        int64  `json:"gameId"`        // 游戏id
	CallbackUrl   string `json:"callbackUrl"`   // 回调地址
	CallbackKey   string `json:"callbackKey"`   // 回调凭证
}
