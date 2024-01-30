# gateway

游戏网关服务

## 前置操作
### 搭建 zookeeper 服务
#### 生成 zookeeper 连接 UAF 信息
```golang
key := configuration.DesKey
src := "{\"backend\":\"127.0.0.1:2181\",\"username\":\"guest\",\"password\":\"guest\"}"
fmt.Println("原文：" + src)
enc, _ := utils.Encrypt([]byte(src), []byte(key))
ioutil.WriteFile("configuration.uaf", []byte(base64.URLEncoding.EncodeToString(enc)), 0644)
fmt.Println("密文：" + base64.URLEncoding.EncodeToString(enc))
dec, _ := utils.Decrypt(enc, []byte(key))
fmt.Println("解码：" + string(dec))
```

#### 环境变量
将生成的密文配置到环境变量`UAF`
> UAF="OVRqX--p2-OICGdUTPu_ZdUw0PpMilPVJdqXbwVOZ_-k8zaY0YHEel8Anhu-DGSzLnHdoMs56IwQFD5lDqgctqWJboHgBSZZ"

或将密文写入文件`configuration.uaf`

## 配置
### 网关业务配置
> PATH: /system/app/game/gateway
```json
{
  "gameList" : [ "jumpjumpjump", "little-game" ] // 后端可连接游戏列表
}
```

### ws配置
> PATH: /system/base/server/ws/10000
```json
{
  "addr" : "tcp://:9009" // 前端 websocket 连接地址
}
```

### tcp配置
> PATH: /system/base/server/tcp/10000
```json
{
  "addr" : ":9800" // 后端服务 tcp 连接地址
}
```

### 配置完成，跑起来
> go run .

### 前端服务通过`WS`连接与网关进行交互
- `sid` 是用户登录凭证
> ws://127.0.0.1:9009?sid=0e87b81c-f5a0-4124-bbc7-065c945e24fc

### 后端服务通过`TCP`连接与网关进行交互
- 配置
> PATH: /system/base/client/tcp/10000
```json
{
  "addr" : "127.0.0.1:9800"
}
```

- 代码样例
```go
var cfg dto.TcpConfig
if err = conf.Clazz("base", "client", "tcp", wire.GateWaySystemId, &cfg); err != nil {
    panic("加载TCP Client 连接配置出错")
}

// request 消息通道
inMsg := make(chan dto.CommonReq, 1)
// response 消息通道
outMsg := make(chan dto.CommonRes, 1)

// 处理 request 消息
go func() {
    wsType := reflect.TypeOf(ws)
    for req := range inMsg {
        if req.Server != dto.GameName {
            logger.Log.Error("called not this server")
            continue
        }
        if _, ok := wsType.MethodByName(req.Msg.Event); ok {
            go reflect.ValueOf(ws).MethodByName(req.Msg.Event).Call([]reflect.Value{reflect.ValueOf(req)})
        } else {
            logger.Log.Error("call method not found")
            continue
        }
    }
}()

// tcp连接
for {
    conn, err := net.Dial("tcp", cfg.Addr)
    if err != nil {
        logger.Log.Errorf("Gateway Tcp Connect Error: %+v", err)
        time.Sleep(time.Second * 1)
        continue
    }
    conn.Write([]byte(fmt.Sprintf("%s\n", dto.GameName))) // 发送消息证明自己是哪个游戏服务

    exitChan := make(chan struct{})
    // 处理 response 消息
    go func(ec <-chan struct{}) {
        for res := range outMsg {
            select {
            case <-ec:
                return
            default:
                resByte, _ := json.Marshal(res)
                _, err := conn.Write(append(resByte, '\n'))
                if err != nil {
                    logger.Log.Errorf("Push Write Error: %+v, Msg: %+v", err, res)
                }
            }
        }
    }(exitChan)

    // 处理 request 消息
    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        var buf = scanner.Bytes()
        logger.Log.Info("recv message: ", string(buf))
        var req dto.CommonReq
        err = json.Unmarshal(buf, &req)
        if err != nil {
            logger.Log.Errorf("unmarshal failed, err: %+v", err)
            continue
        }
        inMsg <- req
    }
    close(exitChan)
}
```