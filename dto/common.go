package dto

import "encoding/json"

type CommonReq struct {
	Server string          `json:"server"`           // 服务
	Event  string          `json:"event"`            // 事件
	Seq    int64           `json:"seq"`              // 请求id
	UserId int64           `json:"userId,omitempty"` // 用户id
	Data   json.RawMessage `json:"data,omitempty"`   // 数据
}

type AuthReq struct {
	Token string `json:"token"`
}

// CommonRes 给客户端的消息
type CommonRes struct {
	Server string          `json:"server"`           // 服务名称
	Event  string          `json:"event"`            // 客户端事件
	Seq    int64           `json:"seq,omitempty"`    // 请求id
	UserId int64           `json:"userId,omitempty"` // 用户id，为 0 发给所有人
	Code   int             `json:"code"`             // 错误码
	Msg    string          `json:"msg,omitempty"`    // 错误信息
	Data   json.RawMessage `json:"data,omitempty"`   // 数据
}
