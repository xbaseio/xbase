# xsnowflake

多服部署时，每个同时运行的进程必须配置不同的 `nodeID`，有效范围为 `0~1023`。

例如：

```toml
# 登录服 1
[snowflake]
nodeID = 1
```

```toml
# 登录服 2
[snowflake]
nodeID = 2
```

服务启动时初始化一次：

```go
if err := xsnowflake.Init(nodeID); err != nil {
    return err
}
```

业务中直接生成：

```go
id, err := xsnowflake.ID()
if err != nil {
    return err
}
```

不要使用随机节点 ID，也不要直接对服务名进行取模；两台服务器一旦得到相同的
`nodeID`，雪花算法就不能保证 ID 唯一。建议由部署配置或统一的节点分配服务维护节点 ID。
