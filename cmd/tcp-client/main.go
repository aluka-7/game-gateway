package main

import (
	"bufio"
	"github.com/aluka-7/game-gateway/tcp"
	pb "github.com/aluka-7/game-gateway/tcp/proto"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"time"
)

const (
	addr      = "127.0.0.1:9800"
	gameAlias = "wingo"
)

func main() {
	for {
		run()
		log.Println("连接断开，5秒后重连...")
		time.Sleep(5 * time.Second)
	}
}

func run() {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println("连接失败:", err)
		return
	}
	defer conn.Close()

	log.Println("✅ 已连接:", addr)

	// 1️⃣ 发送 alias
	_, err = conn.Write([]byte(gameAlias + "\n"))
	if err != nil {
		log.Println("发送 alias 失败:", err)
		return
	}
	log.Println("➡️ 已注册", gameAlias)

	reader := bufio.NewReader(conn)

	// 2️⃣ 启动读协程
	go readLoop(reader)

	// 3️⃣ 定时发送测试包
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		msg := &pb.TcpMessage{
			Server: gameAlias,
			Event:  "result",
			Seq:    time.Now().Unix(),
			UserId: 10001,
			Data:   []byte(`{"name":"hello game client"}`),
		}

		err := sendMessage(conn, msg)
		if err != nil {
			log.Println("❌ 发送失败:", err)
			return
		}

		log.Printf("➡️ 发送:%s data:%s", msg.Event, msg.Data)
	}
}

func sendMessage(conn net.Conn, msg *pb.TcpMessage) error {
	body, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	frame := tcp.EncodeFrame(body)

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write(frame)
	return err
}

func readLoop(reader *bufio.Reader) {
	for {
		payload, err := tcp.ReadFrame(reader)
		if err != nil {
			log.Println("❌ 读取失败:", err)
			return
		}

		packet := new(pb.TcpMessage)
		if err := proto.Unmarshal(payload, packet); err != nil {
			log.Println("❌ 解码失败:", err)
			continue
		}

		log.Printf("⬅️ 收到:server=%s event=%s seq=%d code=%d msg=%s data=%s\n",
			packet.Server,
			packet.Event,
			packet.Seq,
			packet.Code,
			packet.Msg,
			string(packet.Data),
		)
	}
}
