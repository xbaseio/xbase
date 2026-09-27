# xcache 进程内缓存

`xcache` 是当前进程内的并发安全缓存，不依赖 Redis，不读取 Redis 配置，不在服务实例间共享；重启后数据丢失。适合可重新加载的配置、字典等数据，不用于钱包余额、跨实例幂等或分布式锁。

## 使用

```go
import (
    "fmt"
    "time"

    "corebiz/internal/platform/xcache"
)

func example() {
    local := xcache.New(5*time.Minute, time.Minute)
    defer local.Close()

    local.Set("name", "corebiz", xcache.DefaultExpiration)
    local.Set("constant", 42, xcache.NoExpiration)

    if value, found := local.Get("name"); found {
        name, ok := value.(string)
        if ok {
            fmt.Println(name)
        }
    }

    local.Delete("name")
}
```

## 过期与生命周期

- `DefaultExpiration = 0` 表示使用构造时的默认 TTL；`NoExpiration = -1` 表示永不过期。这与 Redis 公共缓存传零表示永不过期的约定不同。
- `New` 的默认 TTL 不大于零时，默认不失效；清理间隔不大于零时，不启动后台清理。
- TTL 从写入时计算，读取和自增自减不会续期。过期条目立即对查询不可见，内存由后台清理、`DeleteExpired`、删除或覆盖回收。
- `Close()` 可重复、并发调用，只发出停止后台清理的信号，不等待正在执行的清理或回调，也不清空或禁止读写。关闭后需要时主动调用 `DeleteExpired()`。
- 缓存没有容量上限或 LRU 淘汰策略；不要无限写入永久键。`ItemCount()` 包含尚未清理的过期条目，`Items()` 仅返回未过期条目的浅拷贝。

## 并发和回调

- 缓存方法保护内部 map；存放的指针、map、slice 等对象不会深拷贝，其内部修改仍需调用方同步。
- `NewFrom` 复制传入 map，保留条目的绝对过期时间；构造期间不要并发修改输入 map。传 nil 可以创建空缓存。
- `OnEvicted` 用于手动删除和过期清理，不用于覆盖或 `Flush`。回调在锁外同步执行，可重入缓存方法；并发删除可能并发调用回调。
- 删除操作使用持锁时捕获的回调。调用 `OnEvicted(nil)` 不会取消已开始删除的一批通知。
- 回调应短小且自行保证并发安全；慢回调会延迟当前删除/清理，panic 不会被缓存吞掉。
- 数值增减保持 Go 原生数值运算语义，不提供金额精度或溢出保护。
- `sharded.go` 仍是未导出的实验分片实现，不改变 `New` 的单 map 行为。

旧的 Gob `Save/Load/SaveFile/LoadFile` 接口保留兼容，不代表可靠持久化；仅加载可信数据，文件保存也不保证原子替换。
