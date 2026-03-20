# 🚪 Gateway

游戏网关服务（WebSocket ↔ TCP 转发层）

Game Gateway 是一个基于 Go 的游戏网关服务，实现 WebSocket 客户端接入、内部 TCP 转发至游戏服务、用户认证（JWT）、心跳检测与多游戏路由，适用于后端实时游戏系统架构。

---

## 📦 功能简介

- 🌐 WebSocket 接入（客户端）
- 🔌 TCP 转发（游戏服务）
- 🔐 用户认证（JWT）
- ❤️ 心跳检测（Ping/Pong）
- 🎮 多游戏服务路由

---

## 🚀 快速开始

### 1️⃣ 前置依赖

#### 🧩 启动 Zookeeper

用于配置中心（存储网关配置）

---

### 2️⃣ 生成 UAF 配置

用于连接 Zookeeper（加密配置）

```go
key := configuration.DesKey
src := "{\"backend\":\"127.0.0.1:2181\",\"username\":\"guest\",\"password\":\"guest\"}"

enc, _ := utils.Encrypt([]byte(src), []byte(key))
cipher := base64.URLEncoding.EncodeToString(enc)

fmt.Println("密文：", cipher)
```

---

### 3️⃣ 配置 UAF

提供两种方式：

#### ✅ 方式一：环境变量（推荐）

```bash
export UAF="你的密文"
```

#### ✅ 方式二：文件

```bash
echo "你的密文" > configuration.uaf
```

---

## ⚙️ 配置说明

---

### 🎮 网关业务配置

📍 路径：`/system/app/game/gateway`

```json
{
  "gameList": ["wingo"]
}
```

> 允许接入的游戏服务列表（TCP）

---

### 🌐 web 配置

📍 路径：`/system/base/server/10000`

```json
{
  "addr" : ":9006",
  "enableLog" : true
}
```

### 🔗 WebSocket 配置

📍 路径：`/system/base/server/ws/10000`

```json
{
  "addr": "tcp://:9009"
}
```

> 客户端连接地址

---

### 🧠 Redis 配置

📍 路径：`/system/base/cache/10000`

```json
{
  "provider": "redis",
  "database": "0",
  "ping": "true",
  "host": "127.0.0.1",
  "port": "6379",
  "password": "123456"
}
```

---

### 🔌 TCP 配置（游戏服务）

📍 路径：`/system/base/server/tcp/10000`

```json
{
  "addr": ":9800"
}
```

> 游戏服务连接入口（内部通信）

---

## ▶️ 启动服务

```bash
go run .
```

---

## 🔐 鉴权与协议约定

### WebSocket 鉴权

- 客户端连接后需在 **5 秒内**发送 `system/auth` 消息，否则连接会被断开。
- `data.token` 格式必须为：`Bearer <JWT>`。

示例：

```json
{
  "server": "system",
  "event": "auth",
  "seq": 1740000000,
  "data": {
    "token": "Bearer <your-jwt>"
  }
}
```

### 心跳

- 客户端应定时发送 `system/ping`。
- 服务端回复 `system/pong`。
- **30 秒**未更新心跳会被断开。

### TCP 首包约定

- 游戏服务连入 TCP 后，首包必须是 `alias + "\n"`（例如：`wingo\n`）。
- 后续消息使用 `length-frame + protobuf`。

---

## 🧪 测试

### 🔗 WebSocket 客户端

```bash
go run ./cmd/ws-client
```

---

### 🔌 TCP 客户端

```bash
go run ./cmd/tcp-client
```

---

## 🧠 架构说明

```
[ Client ]
    │  WebSocket
    ▼
[ Gateway ]
    │  TCP
    ▼
[ Game Server ]
```

---

## ⚠️ 注意事项

- 游戏服务需先连接 TCP 并发送 alias（首包）
- alias 必须在 `gameList` 中配置
- 认证超时（5 秒）会被断开
- 心跳超时（30 秒）自动断开连接
- TCP 使用 length-frame + protobuf 协议

---

## 📌 常用命令

```bash
# 启动网关
go run .

# 运行 WS 测试客户端
go run ./cmd/ws-client

# 运行 TCP 测试客户端
go run ./cmd/tcp-client
```

## 📊 监控
Prometheus 指标地址：
```json
http://ip:7070/metrics
```
