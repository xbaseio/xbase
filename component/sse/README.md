# SSE Component

`component/sse` is a mount-only SSE module for the existing Fiber-based HTTP server.

It does not open its own port. Mount it onto `component/http` and use the proxy to publish events.

## Minimal Usage

```go
httpSrv := http.NewServer(
    http.WithAddr(":8080"),
)

sseSrv := sse.NewServer(
    sse.WithPath("/events"),
)

sseSrv.MountHTTP(httpSrv.Proxy())

httpSrv.Proxy().Router().Get("/publish", func(ctx http.Context) error {
    n := sseSrv.Proxy().Publish("lobby", sse.Event{
        Event: "message",
        Data:  map[string]any{"text": "hello"},
    })
    return ctx.Success(map[string]any{"delivered": n})
})
```

Subscribe from the browser:

```js
const es = new EventSource("/events?topic=lobby");
es.addEventListener("message", (e) => {
  console.log(e.data);
});
```

## Defaults

- stream path: `/events`
- health path: `/healthz`
- topic query key: `topic`
- heartbeat interval: `15s`

## Config Keys

- `etc.sse.name`
- `etc.sse.path`
- `etc.sse.healthPath`
- `etc.sse.topicQueryKey`
- `etc.sse.clientBuffer`
- `etc.sse.heartbeatInterval`
