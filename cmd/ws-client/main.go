package main

import (
	"fmt"
	"github.com/aluka-7/game-gateway/ws"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"sync"
	"time"
)

var (
	target    = "ws://127.0.0.1:9009"
	heartbeat = 10 * time.Second
)

func main() {
	u, _ := url.Parse(target)

	log.Println(fmt.Sprintf("Connecting to: %s", u.String()))

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer conn.Close()
	var writeMu sync.Mutex
	sendJSON := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(v)
	}

	log.Println("✅ 已连接")

	// ========= 1️⃣ 生成 token =========
	claims := ws.UserClaims{
		User: ws.User{
			Id: 10001,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "test-user",
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("kX9Gxcd1-@0eV-*1"))
	if err != nil {
		log.Fatal("token生成失败:", err)
	}

	// ========= 2️⃣ 发送 auth =========
	err = sendJSON(map[string]interface{}{
		"server": "system",
		"event":  "auth",
		"seq":    time.Now().Unix(),
		"data": map[string]interface{}{
			"token": "Bearer " + token,
		},
	})
	if err != nil {
		log.Fatal("auth发送失败:", err)
	}

	log.Println("➡️ 发送:auth")

	// ========= 3️⃣ 心跳 =========
	go func() {
		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()

		for range ticker.C {
			err := sendJSON(map[string]interface{}{
				"server": "system",
				"event":  "ping",
				"seq":    time.Now().Unix(),
			})
			if err != nil {
				log.Println("心跳发送失败:", err)
				return
			}
			log.Println("➡️ 发送:ping")
		}
	}()

	// ========= 4️⃣ 测试业务消息 =========
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			err := sendJSON(map[string]interface{}{
				"server": "wingo",
				"event":  "test",
				"seq":    time.Now().Unix(),
				"data": map[string]interface{}{
					"msg": "hello game server",
				},
			})
			if err != nil {
				log.Println("业务消息发送失败:", err)
				return
			}

			log.Printf("➡️ 发送测试消息")
		}
	}()

	// ========= 5️⃣ 读取消息 =========
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			log.Println("❌ 读取失败:", err)
			return
		}

		log.Printf("⬅️ 收到:type=%d payload=%s\n",
			messageType,
			string(payload),
		)
	}
}
