# xbase Node.js 测试客户端

用于连接 **xbase Gate** 的轻量测试工具，协议与 Go 端 `packet` 包一致（默认 **Big Endian**）。

## 协议说明

每帧二进制结构：

```
[4B totalLen][4B dataBit=0][4B GameID][4B MessageID][4B Seq][Body...]
```

- `totalLen` = 20 + len(Body)，不含最外层 4 字节长度前缀
- **GameID**：`0` 为 Gate 控制消息，`1` 为大厅，`2+` 由 Gate 选择对应游戏 Node
- **MessageID**：Node 内业务 Handler 用（第二级路由）
- 字节序默认 **BE**，与 `etc.packet.byteOrder=big` 一致；若服务端配了 `little`，请加 `--endian LE`

## 安装

```bash
cd examples/nodejs-client
npm install
```

## 使用

### 单次发送（TCP）

```bash
node src/index.js \
  --host 127.0.0.1 \
  --port 3553 \
  --game 1 \
  --message 1001 \
  --body hello
```

### WebSocket

Gate 使用 WS 监听时：

```bash
node src/index.js \
  --ws \
  --host 127.0.0.1 \
  --port 3653 \
  --game 1 \
  --message 1001 \
  --json '{"action":"ping"}'
```

### 压测 / 连发

```bash
node src/index.js --game 1 --message 1001 --body ping --count 100 --interval 10
```

### 交互模式

```bash
node src/index.js --interactive --game 1 --listen 0
```

输入格式：`<messageID> [body]`，空行退出。

## 参数一览

| 参数 | 默认 | 说明 |
|------|------|------|
| `--host` | `127.0.0.1` | Gate 监听地址 |
| `--port` | `3553` | Gate 监听端口 |
| `--ws` | - | 使用 WebSocket |
| `--endian` | `BE` | `BE` / `LE` |
| `--game` | `1` | 业务 GameID，默认为大厅 |
| `--message` | - | MessageID |
| `--body` | `''` | UTF-8 body |
| `--json` | - | JSON 字符串 body |
| `--count` | `1` | 发送条数 |
| `--interval` | `0` | 发送间隔 ms |
| `--listen` | `5` | 发送后监听下行秒数 |
| `--interactive` | - | 交互模式 |

## 与服务端联调

1. 启动 Gate（TCP 或 WS），确认 **对外 network 端口**（不是 cluster link 端口）
2. Node 绑定统一的 `MessageDispatcher`，由业务代码分发 `MessageID`
3. 客户端 `--game` 与 Node 的 `WithGameID` 一致
4. `GameID=0` 的登录和认证由 Gate 上层 dispatcher 处理；完成校验后调用 `ctx.Bind(uid)`，存在 `GameID=1` 大厅时会自动绑定大厅

## 目录

```
examples/nodejs-client/
├── package.json
├── README.md
└── src/
    ├── packet.js   # 编解码
    ├── client.js   # TCP / WS 连接
    └── index.js    # CLI 入口
```
