# xbase

`xbase` 是一套面向实时业务和游戏服务器场景的 Go 分布式框架。

它把常见的几个问题拆开处理：

- 对外连接由 `Gate` 负责
- 业务逻辑由 `Node` 负责
- RPC 服务由 `Mesh` 负责
- 服务发现、用户定位、版本切换、灰度路由、事件总线这些基础能力由框架统一提供

如果你想快速理解这套框架，可以先抓住一句话：

`Client -> Gate -> Node -> Mesh/RPC`

---

## 适合什么场景

这套框架更适合下面这类服务：

- 游戏大厅、房间服、战斗服
- 有长连接入口的实时业务
- 需要按用户做状态绑定的服务
- 需要滚动发布、灰度放量、版本切换的集群

---

## 核心角色

### Gate

`cluster/gate`

Gate 是对外入口，主要负责：

- 管理客户端连接和会话
- 收包、解包、登录
- 按 `GameID` 把消息转发给对应 Node 组
- 在需要时绑定用户和 Gate 的关系
- 订阅集群路由信息和灰度策略

你可以把它理解成“连接层 + 转发层”。

### Node

`cluster/node`

Node 是业务执行层，主要负责：

- 把业务消息交给绑定的消息分发函数
- 提供消息上下文、Actor 和集群调用能力
- 维护有状态路由
- 绑定用户和 Node 的关系
- 可选内置 Actor 调度
- 可选暴露 RPC 服务给别的进程调用

你可以把它理解成“游戏逻辑服 / 业务逻辑服”。

### Mesh

`cluster/mesh`

Mesh 是纯 RPC 服务进程，主要负责：

- 注册和暴露 gRPC / rpcx 服务
- 被 Node 或其他 Mesh 发现并调用
- 参与版本过滤和滚动发布

你可以把它理解成“独立微服务进程”。

### Container

`container.go`

所有组件都挂在 `xbase.Container` 里统一管理生命周期：

```go
container := xbase.NewContainer()
container.Add(comp1, comp2, comp3)
container.Serve()
```

执行顺序是：

`Init -> Start -> 等待退出信号 -> Close -> Destroy`

---

## 一条消息怎么走

框架里消息是两级路由：

1. Gate 按 `GameID` 区分消息归属：`0` 是 Gate 控制消息，`1` 是大厅，`2+` 是对应游戏
2. Gate 控制消息直接交给 Gate 上层 dispatcher，其余消息按 `GameID` 选择 Node 组
3. Node 把消息交给业务绑定的 dispatcher

也就是：

`Client -> Gate -> GameID -> (GateDispatcher | Node -> MessageDispatcher) -> Handler/Actor`

对应的数据结构是 `cluster.Message` / `packet.Message` 这一套。

### 为什么这样设计

这样拆以后：

- Gate 不需要理解业务消息，只负责选业务组
- Node 不持有业务路由表，业务可自由选择生成代码、模块路由、Actor 或脚本分发
- 集群扩容时更容易按游戏、房间、业务组拆分

Gate 控制消息可以绑定上层处理函数：

```go
gate.NewGate(
	gate.WithMessageDispatcher(func(ctx gate.Context) {
		// 处理 GameID=0 的 Gate 消息
		// 完成登录校验后调用 ctx.Bind(uid)
	}),
)
```

---

## 路由和状态

### 无状态路由

普通消息没有绑定关系时，会走 `Dispatcher` 的负载均衡：

- `random`
- `rr`
- `wrr`

### 有状态路由

如果某类业务需要“用户固定落到某个 Node”，就会用 Locator 记录：

`uid -> node`

这样后续请求会优先打到同一台 Node。

这类场景常见于：

- 房间服
- 战斗服
- 单用户状态机

### 绑定消息分发函数

Node 推荐只绑定一个业务消息入口，普通消息不需要逐条向框架注册：

```go
n.Proxy().BindMessageDispatcher(func(ctx node.Context) {
	switch ctx.MessageID() {
	case 1001:
		handlePing(ctx)
	case 2001:
		handleEnterRoom(ctx)
	default:
		handleUnknown(ctx)
	}
})
```

业务可以在这个入口后接自己的协议生成器、模块路由、Actor 或脚本系统，Node 不限制具体实现。

只有消息需要特殊集群调度策略时才声明策略，例如：

```go
n.Proxy().SetRoutePolicy(2001, node.StatefulPolicy)
```

这不是注册业务 handler；没有声明策略的消息同样会进入 dispatcher。

### Internal / Stateful

Node 只关心影响集群投递的策略：

- `Internal`：仅集群内部可达
- `Stateful`：需要用户定位到固定 Node

登录态和鉴权属于 Gate 边界。Node 接收集群内部流量，不重复做逐消息鉴权。

---

## 服务发现和用户定位

### Registry

`registry/*`

Registry 负责服务注册和发现。当前仓库内有：

- Consul
- Nacos
- etcd 风格接口抽象

服务实例会注册这些信息：

- `Kind`
- `Alias`
- `GameID`
- `Version`
- `Endpoint`
- `Routes`
- `Events`
- `Metadata`

### Locator

`locate/*`

Locator 负责用户位置绑定。常见关系有：

- `uid -> gate`
- `uid -> node`

这样 Gate、Node、Mesh 都能知道一个用户当前应该路由到哪里。

默认示例使用的是 Redis Locator。

---

## 版本和灰度

这套框架现在支持两种常见控制维度：

### 版本过滤

服务实例注册时会带 `Version`。

新流量默认只会进入同组里的最高版本，旧版本实例可以在一段 `retireDelay` 内继续承接已经绑定的老用户，用来做滚动发布。

### 服务状态

Node 还可以带 `serviceStatus`：

- `normal`
- `gray`
- `test`

当前 Gate 的登录路由已经支持：

- 普通用户进 `normal`
- 灰度白名单或灰度比例命中进 `gray`
- 测试白名单进 `test`
- 当高层状态不可用时，按 `test -> gray -> normal` 回退

这部分在示例里有完整演示。

---

## RPC 和 Mesh

如果 Node 或独立 Mesh 挂了 transporter，就可以把服务注册成 RPC 服务。

当前仓库支持：

- `transport/grpc`
- `transport/rpcx`

常见调用方式是：

- `direct://ip:port`
- `direct://instance-id`
- `discovery://service-name`

所以 Node 既可以只做消息处理，也可以顺手暴露一部分 RPC 服务给别的进程调用。

---

## 事件总线

`eventbus/*`

框架内有统一事件总线抽象，当前仓库里有：

- Redis
- NATS
- Kafka

当前示例里已经用它做了一个很典型的事情：

- Gate 订阅灰度策略事件
- 其他服务或脚本发布策略事件
- Gate 在线更新灰度/测试路由策略

---

## 最小上手方式

最直接的学习路径不是从零写代码，而是先跑示例：

### 1. 看示例目录

- [examples/cluster-demo/README.md](examples/cluster-demo/README.md)
- [examples/nodejs-client/README.md](examples/nodejs-client/README.md)

### 2. 启动依赖

`cluster-demo` 需要：

- Redis
- Consul
- NATS

### 3. 启动示例进程

```bash
go run ./examples/cluster-demo/node -etc ./examples/cluster-demo/etc/node-normal.toml
go run ./examples/cluster-demo/node -etc ./examples/cluster-demo/etc/node-gray.toml
go run ./examples/cluster-demo/node -etc ./examples/cluster-demo/etc/node-test.toml
go run ./examples/cluster-demo/gate -etc ./examples/cluster-demo/etc/gate.toml
go run ./examples/cluster-demo/mesh -etc ./examples/cluster-demo/etc/mesh.toml
```

### 4. 再看代码入口

- `examples/cluster-demo/gate/main.go`
- `examples/cluster-demo/node/main.go`
- `examples/cluster-demo/mesh/main.go`

这里基本能看清楚：

- 组件怎么组装
- Registry / Locator / Transporter 怎么注入
- 路由怎么注册
- 调试服务怎么挂
- 灰度策略事件怎么接

---

## 读代码建议

如果你是第一次进这个仓库，我建议按这个顺序看：

1. `container.go`
2. `cluster/cluster.go`
3. `cluster/gate`
4. `cluster/node`
5. `internal/link`
6. `internal/dispatcher`
7. `registry` 和 `locate`
8. `transport`
9. `examples/cluster-demo`

这样会比直接从 `internal` 深处开始看轻松很多。

---

## 目录概览

```text
xbase/
├─ cluster/         Gate / Node / Mesh / Client
├─ component/       通用组件
├─ eventbus/        事件总线实现
├─ locate/          用户定位
├─ registry/        服务注册发现
├─ transport/       gRPC / rpcx
├─ network/         TCP / WS / KCP 等网络层
├─ session/         会话管理
├─ internal/        Dispatcher / Linker / Transporter 等内部实现
├─ examples/        最小示例和调试示例
└─ container.go     组件容器入口
```

---

## 当前建议

如果你准备基于这套框架继续开发，建议优先保留这几个边界：

- Gate 只做连接、登录、转发、入口侧策略
- Node 只做消息路由和业务逻辑
- Mesh 只做 RPC 服务
- 灰度、事件、配置更新尽量用统一基础设施，不要散在业务里

这样后面做扩容、拆服、灰度和排障都会轻松很多。
