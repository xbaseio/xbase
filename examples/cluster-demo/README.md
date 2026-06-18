# cluster-demo

这套示例给你一套最小可跑的 `gate / node / mesh` 启动方式，并把“测试服白名单”配置一起放进来了。

## 依赖

启动前先准备：

1. Redis: `127.0.0.1:6379`
2. Consul: `127.0.0.1:8500`

## 启动命令

先启动正式 Lobby Node：

```bash
go run ./examples/cluster-demo/node -etc ./examples/cluster-demo/etc/node-normal.toml
```

再启动测试 Lobby Node：

```bash
go run ./examples/cluster-demo/node -etc ./examples/cluster-demo/etc/node-test.toml
```

启动 Gate：

```bash
go run ./examples/cluster-demo/gate -etc ./examples/cluster-demo/etc/gate.toml
```

启动 Mesh：

```bash
go run ./examples/cluster-demo/mesh -etc ./examples/cluster-demo/etc/mesh.toml
```

## 配置说明

核心演示点在这几个键：

```toml
[cluster.node]
name = "lobby"
gameID = 0
version = "1.0.0"
serviceStatus = "normal"
```

```toml
[cluster.node]
name = "lobby"
gameID = 0
version = "2.0.0"
serviceStatus = "test"
```

```toml
[cluster.gate]
testWhitelist = [10001, 10002]
```

含义是：

- 普通用户只会进 `serviceStatus = "normal"` 的最高版本
- 白名单用户可以进入 `serviceStatus = "test"` 的最高版本
- 这里 `node-normal.toml` 是正式服 `1.0.0`
- `node-test.toml` 是测试服 `2.0.0`

## 路由示例

`node/main.go` 里注册了两个路由：

1. `1001`: 普通回显路由
2. `2001`: `AuthorizedRoute`，要求用户先完成 Gate 登录
3. `3001`: `AuthorizedRoute`，会在 `node` 内部通过 RPC 调 `user.rpc/GetProfile`

返回里会带上：

- 当前命中的 `node` 名称
- 当前 `version`
- 当前 `serviceStatus`
- 当前请求的 `uid`

这样你一眼就能看出消息到底打到了正式服还是测试服。

## RPC 示例

这套示例里，`node` 额外挂了一个 `rpcx` 服务：

```go
n.Proxy().AddServiceProvider("user.rpc", "user.rpc", &UserRPC{})
```

配置走的是：

```toml
[transport.rpcx.server]
addr = ":20001"
```

消息路由里通过服务发现调用：

```go
cli, err := ctx.NewMeshClient("discovery://user.rpc")
err = cli.Call(ctx.Context(), "user.rpc", "GetProfile", req, resp)
```

含义是：

- `AddServiceProvider` 把 `user.rpc` 挂到当前 `node`
- `discovery://user.rpc` 通过注册中心找可用服务实例
- `3001` 路由会调一次 `user.rpc/GetProfile`，再把 RPC 结果回给客户端

## 登录约定

`gate.toml` 里已经开启了登录路由：

```toml
[cluster.gate.login]
messageID = 1000

[cluster.gate.jwt]
secretKey = "cluster-demo-secret"
identityKey = "uid"
signAlgorithm = "HS256"
```

也就是说：

1. 客户端发 `messageID = 1000` 的登录消息
2. Token 里要有 `uid` claim
3. `uid` 在 `testWhitelist` 里时，会优先路由到测试服最高版本

## 你可以怎么扩展

如果你想继续压测或做灰度，可以直接照着这个思路加：

- 多个 `normal` 节点，版本改成 `1.0.0 / 1.1.0 / 1.2.0`
- 多个 `test` 节点，版本改成 `2.0.0 / 2.1.0`
- 多个业务组，把 `cluster.node.name` 改成 `lobby`、`battle`、`chat`

## 监控与日志

这套架构上线前，建议至少把下面几类监控补齐。

### 1. Gate 侧

重点看连接和入口路由是否正常：

- 当前连接数
- 每秒新连接数 / 断开数
- 登录成功数 / 失败数
- 未授权消息拦截数
- 收包队列积压
- 投递到 node 的成功数 / 失败数
- 按 `gameID + messageID` 统计的消息量

重点日志字段建议统一带上：

- `uid`
- `cid`
- `gid`
- `gameID`
- `messageID`
- `seq`
- `version`
- `serviceStatus`
- `targetNode`
- `err`

最关键的日志点：

- 登录请求进入
- JWT 校验失败
- `BindGate` 成功 / 失败
- 大厅节点选择结果
- 未授权路由被拦截
- 投递 node 失败

### 2. Node 侧

重点看状态服是否过载、是否打错节点：

- 每个 node 实例当前在线人数
- 每秒请求数
- 按 `messageID` 统计的处理耗时
- 请求队列积压
- 事件队列积压
- Actor / 房间数量
- `BindNode` / `UnbindNode` 次数
- 丢包或超时投递次数

统一日志字段建议：

- `uid`
- `gid`
- `nid`
- `nodeName`
- `gameID`
- `messageID`
- `version`
- `serviceStatus`
- `routeType`
- `latencyMs`
- `err`

最关键的日志点：

- `Connect` / `Disconnect`
- `BindNode` / `UnbindNode`
- 有状态路由命中结果
- 路由未注册
- 请求队列满被丢弃
- 战斗或房间创建 / 销毁

### 3. RPC / Mesh 侧

重点看微服务调用是否稳定：

- 每个 RPC 服务的 QPS
- 每个 RPC 方法的成功率
- 每个 RPC 方法的 P50 / P95 / P99 耗时
- 超时次数
- 服务发现失败次数
- 目标实例不可用次数

统一日志字段建议：

- `service`
- `method`
- `target`
- `targetInstance`
- `callerNode`
- `uid`
- `version`
- `serviceStatus`
- `latencyMs`
- `err`

最关键的日志点：

- 服务注册成功 / 失败
- `NewMeshClient` 创建失败
- `discovery://service_name` 解析不到实例
- RPC 调用超时
- RPC 返回业务错误

### 4. Registry / Locator

这是这套架构最容易被忽略、但排障最关键的一层：

- 注册中心可用性
- 当前 gate / node / mesh 实例数
- 实例状态变更次数
- 实例按 `version + serviceStatus` 分布
- 用户 Gate 绑定数量
- 用户 Node 绑定数量
- Bind / Unbind 失败次数
- 脏绑定数量

最关键的日志点：

- 实例注册 / 反注册
- 实例刷新状态失败
- `LocateGate` / `LocateNode` 失败
- 用户绑定缺失
- 用户绑定到不存在节点

### 5. 灰度相关

因为你这套已经有 `version + serviceStatus + testWhitelist`，所以建议单独做灰度监控：

- `normal` / `test` 流量占比
- 白名单用户命中 `test` 的比例
- 普通用户误进 `test` 的次数
- 不同版本实例的在线人数
- 不同版本实例的错误率

最关键的日志点：

- 登录后选中的目标实例
- 实例被过滤的原因
- 是否命中白名单
- 走的是 `normal` 还是 `test`

推荐至少把下面这些字段打出来：

- `uid`
- `gameID`
- `nodeName`
- `nodeID`
- `version`
- `serviceStatus`
- `allowTestService`
- `whitelistHit`

### 6. 排障顺序

线上遇到“用户消息没到预期节点”时，建议按这个顺序查：

1. 看 Gate 是否收到消息
2. 看用户是否完成登录并拿到 UID
3. 看 `BindGate` 是否成功
4. 看是否命中白名单
5. 看路由时选中的 `gameID / nodeName / version / serviceStatus`
6. 看 Locator 里用户当前绑定到哪个 Node
7. 看目标 Node 是否还在注册中心里
8. 看目标 Node 是否是同组最高版本
9. 看 Node 是否真的收到了消息
10. 如果经过 RPC，再看 RPC 调用是否超时或找不到实例

### 7. 最低落地要求

如果现在时间有限，最少先把下面这些补上：

- 所有关键日志统一带 `uid/gid/nid/gameID/messageID/version/serviceStatus`
- 登录成功、选路结果、`BindGate`、`BindNode`、RPC 调用失败必须打日志
- 每个 `messageID` 的计数和耗时统计
- 每个 RPC 方法的成功率和耗时统计
- 一个按 `uid` 查询当前 Gate/Node 归属的调试接口

这样即使还没接完整监控平台，线上出问题时也至少能快速还原用户链路。

## Debug API

The examples now start a small debug HTTP server in each process:

- gate: `:28001`
- node-normal: `:28002`
- node-test: `:28003`
- mesh: `:28004`

Useful endpoints:

```bash
curl http://127.0.0.1:28002/healthz
curl "http://127.0.0.1:28002/debug/user?uid=10001&node=lobby"
curl "http://127.0.0.1:28002/debug/services?name=node"
curl "http://127.0.0.1:28002/debug/services?name=mesh"
```

The example code also emits structured logs for:

- node connect / disconnect
- route parse and reply failures
- route latency
- rpc client create failures
- rpc call success / failure
- debug API lookups
