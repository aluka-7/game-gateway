# 🚪 Gateway

Game Gateway Service (WebSocket ↔ TCP Forwarding Layer)

Game Gateway is a Go-based game gateway service that implements WebSocket client access, internal TCP forwarding to the game service, user authentication (JWT), heartbeat detection, and multi-game routing. It is suitable for backend real-time game system architectures.

---

## 📦 Feature Overview

- 🌐 WebSocket Access (Client)
- 🔌 TCP Forwarding (Game Service)
- 🔐 User Authentication (JWT)
- ❤️ Heartbeat Detection (Ping/Pong)
- 🎮 Multi-Game Service Routing

---

## 🚀 Quick Start

### 1️⃣ Prerequisites

#### 🧩 Start Zookeeper

Used for configuration center (storage gateway configuration)

---

### 2️⃣ Generate UAF Configuration

Used for connecting to Zookeeper (encrypted configuration)

```go
key := configuration.DesKey
src := "{\"backend\":\"127.0.0.1:2181\",\"username\":\"guest\",\"password\":\"guest\"}"

enc, _ := utils.Encrypt([]byte(src), []byte(key))
cipher := base64.URLEncoding.EncodeToString(enc)

fmt.Println("ciphertext：", cipher)
```

---

### 3️⃣ Configure UAF

Two methods are provided:

#### ✅ Method 1: Environment Variables (Recommended)

```bash
export UAF="your ciphertext"
```

#### ✅ Method 2: File

```bash
echo "your ciphertext" > configuration.uaf
```

---

## ⚙️ Configuration Instructions

---

### 🎮 Gateway Service Configuration

📍 Path: `/system/app/game/gateway`

```json
{
  "gameList": ["wingo"]
}
```

> List of allowed game services (TCP)

---

### 🌐 Web Configuration

📍 Path: `/system/base/server/10000`

```json
{
  "addr" : ":9006",
  "enableLog" : true
}
```

### 🔗 WebSocket Configuration

📍 Path: `/system/base/server/ws/10000`

```json
{
  "addr": "tcp://:9009"
}
```

> Client Connection Address

---

### 🧠 Redis Configuration

📍 Path: `/system/base/cache/10000`

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

### 🔌 TCP Configuration (Game Server)

📍 Path: `/system/base/server/tcp/10000`

```json
{
  "addr": ":9800"
}
```

> Game Service Connection Entry Point (Internal Communication)

---

## ▶️ Start the Service

```bash
go run .

```

---

## 🔐 Authentication and Protocol Conventions

### WebSocket Authentication

- After connecting, the client must send a `system/auth` message within **5 seconds**, otherwise the connection will be closed.
- The `data.token` format must be: `Bearer <JWT>`.

Example:

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

### Heartbeat

- The client should send `system/ping` periodically.
- The server replies with `system/pong`.
- The connection will be closed if the heartbeat is not updated within **30 seconds**.

### TCP First Packet Conventions

- After the game service connects to TCP, the first packet must be `alias + "\n"` (e.g., `wingo\n`). - Subsequent messages will use `length-frame + protobuf`.

---

## 🧪 Testing

### 🔗 WebSocket Client

```bash
go run ./cmd/ws-client
```

---

### 🔌 TCP Client

```bash
go run ./cmd/tcp-client
```

---

## 🧠 Architecture Description

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

## ⚠️ Notes

- The game service must first connect to TCP and send an alias (first packet)
- The alias must be configured in `gameList`
- Authentication timeout (5 seconds) will result in disconnection
- Heartbeat timeout (30 seconds) will automatically disconnect
- TCP uses length-frame + protobuf Protocol

---

## 📌 Common Commands

```bash
# Start the gateway
go run .

# Run the WS test client
go run ./cmd/ws-client

# Run the TCP test client
go run ./cmd/tcp-client
```

## 📊 Monitoring
Prometheus metrics address:

```json
http://ip:7070/metrics
```