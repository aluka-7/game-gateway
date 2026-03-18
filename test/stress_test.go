package test

import (
	"github.com/aluka-7/game-gateway/ws"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var (
	target = "ws://127.0.0.1:9009"

	totalConn    = 30000 // 连接数
	dialRate     = 200   // 每秒建立连接数（防止瞬间打爆）
	heartbeat    = 1 * time.Second
	testDuration = 120 * time.Second

	successConn int64
	failedConn  int64
	sendCount   int64
)

func TestStress(t *testing.T) {
	u, _ := url.Parse(target)

	var wg sync.WaitGroup

	ticker := time.NewTicker(time.Second / time.Duration(dialRate))
	defer ticker.Stop()

	log.Println("Start connecting...")

	for i := 0; i < totalConn; i++ {
		<-ticker.C

		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runClient(u, id)
		}(i)
	}

	go printStats()

	wg.Wait()

	time.Sleep(testDuration)
	log.Println("Test finished")
}

func runClient(u *url.URL, id int) {
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		atomic.AddInt64(&failedConn, 1)
		return
	}

	atomic.AddInt64(&successConn, 1)

	// 生成token
	claims := ws.UserClaims{
		User: ws.User{
			Id: int64(id + 1),
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "1234567890",
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("kX9Gxcd1-@0eV-*1"))

	// auth
	err = conn.WriteJSON(map[string]interface{}{
		"server": "system",
		"event":  "auth",
		"seq":    123456,
		"data": map[string]interface{}{
			"token": "Bearer " + token,
		},
	})
	if err != nil {
		conn.Close()
		return
	}

	// 心跳 goroutine
	go func() {
		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := conn.WriteJSON(map[string]interface{}{
					"server": "system",
					"event":  "ping",
					"seq":    123456,
				})
				if err != nil {
					return
				}
				atomic.AddInt64(&sendCount, 1)
			}
		}
	}()

	// 读消息（防止阻塞）
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			return
		}
	}
}

func printStats() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastSend int64

	for range ticker.C {
		currentSend := atomic.LoadInt64(&sendCount)
		qps := float64(currentSend-lastSend) / 5.0
		lastSend = currentSend

		log.Printf("Conn: success=%d fail=%d | QPS=%.4f\n",
			atomic.LoadInt64(&successConn),
			atomic.LoadInt64(&failedConn),
			qps,
		)
	}
}
