# cluster-demo

这个示例现在演示一套完整的 `normal / gray / test` 路由模型：

- `normal`：正式流量
- `gray`：灰度放量
- `test`：内部测试 / 强白名单

Gate 只负责订阅并应用灰度策略，不负责发布。策略更新通过独立的 `eventbus` 事件发送，别的服务或独立脚本都可以发。

## 依赖

启动前准备：

1. Redis：`127.0.0.1:6379`
2. Consul：`127.0.0.1:8500`
3. NATS：`nats://127.0.0.1:4222`

## 启动顺序

启动正式 Lobby Node：

```bash
go run ./examples/cluster-demo/node -etc ./examples/cluster-demo/etc/node-normal.toml
```

启动灰度 Lobby Node：

```bash
go run ./examples/cluster-demo/node -etc ./examples/cluster-demo/etc/node-gray.toml
```

启动测试 Lobby Node：

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

## 当前路由规则

`gate.toml` 里有这几个关键项：

```toml
[cluster.gate]
grayWhitelist = [10003]
grayTrafficPercent = 20
grayTrafficSalt = "cluster-demo-lobby"
testWhitelist = [10001, 10002]
```

含义是：

- `10001`、`10002` 优先进入 `test`
- `10003` 强制进入 `gray`
- 其他用户按稳定哈希命中 `20% gray`
- 未命中灰度的普通用户进入 `normal`

状态回退顺序是：

- `test -> gray -> normal`
- `gray -> normal`
- `normal -> normal`

也就是测试服没开时，测试用户还能自动落到灰度或正式，不会直接报错。

## 节点配置

三个示例节点分别是：

- [node-normal.toml](/D:/go-work/src/xbase/examples/cluster-demo/etc/node-normal.toml)
- [node-gray.toml](/D:/go-work/src/xbase/examples/cluster-demo/etc/node-gray.toml)
- [node-test.toml](/D:/go-work/src/xbase/examples/cluster-demo/etc/node-test.toml)

对应状态分别为：

- `normal`：`1.0.0`
- `gray`：`1.5.0`
- `test`：`2.0.0`

## 在线热更新

Gate 只订阅策略更新事件：

- 全局 `eventbus` 由上层按需挂载 `component/eventbus.Global`
- 订阅函数：`component/eventbus.SubscribeServiceStatusPolicy(...)`
- 订阅主题：`cluster.gate.service_status_policy`

示例 Gate 进程启动时会初始化 NATS `eventbus`，然后挂载订阅组件。

发布动作放在独立示例里：

```bash
go run ./examples/cluster-demo/policy-publisher -etc ./examples/cluster-demo/etc/gate.toml
```

这个示例会把当前 `gate.toml` 里的灰度策略发布到 `eventbus`，Gate 收到后会在线生效，不需要重启。

如果你要在别的服务里发，也直接用：

```go
payload := eventbuscomponent.ServiceStatusPolicyEvent{
    GateNames: []string{"demo-gate"},
    Policy: eventbuscomponent.ServiceStatusPolicy{
        GrayTrafficPercent: 30,
        GrayWhitelist:      []int64{10003, 10004},
        TestWhitelist:      []int64{10001},
    },
}

eventbuscomponent.PublishServiceStatusPolicy(&payload)
```

## 路由与 RPC 示例

`node/main.go` 里保留了 3 个演示路由：

1. `1001`：普通回显
2. `2001`：登录后才能访问
3. `3001`：登录后访问，并在节点内部调用 `user.rpc/GetProfile`

返回体里会带上：

- `from`
- `uid`
- `version`
- `serviceStatus`

这样很容易看出请求最后打到了 `normal / gray / test` 哪一层。

## Debug API

每个进程都带一个小的调试 HTTP 服务：

- gate: `:28001`
- node-normal: `:28002`
- node-gray: `:28003`
- node-test: `:28004`
- mesh: `:28005`

常用接口：

```bash
curl http://127.0.0.1:28002/healthz
curl "http://127.0.0.1:28002/debug/user?uid=10001&node=lobby"
curl "http://127.0.0.1:28002/debug/services?name=node"
curl "http://127.0.0.1:28002/debug/services?name=mesh"
```

## 关键日志

Gate 选路日志现在会带上这些字段：

- `targetStatus`
- `selectedStatus`
- `reason`
- `grayWhitelistHit`
- `testWhitelistHit`
- `grayTrafficHit`
- `grayTrafficBucket`
- `nodeID`
- `nodeName`
- `version`

这几项基本够你排灰度是否命中、为什么命中、最终落到了哪一层。
