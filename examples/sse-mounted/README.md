# sse-mounted

This example shows the recommended usage: mount `component/sse` onto the existing Fiber HTTP server.

## Run

```bash
go run ./examples/sse-mounted
```

## Subscribe

Open in the browser console:

```js
const es = new EventSource("http://127.0.0.1:8088/events?topic=lobby");
es.addEventListener("message", (e) => console.log("message", e.data));
es.addEventListener("broadcast", (e) => console.log("broadcast", e.data));
```

## Publish

Topic publish:

```bash
curl "http://127.0.0.1:8088/publish?topic=lobby&text=hello"
```

Broadcast:

```bash
curl "http://127.0.0.1:8088/broadcast?text=world"
```

## Health

```bash
curl http://127.0.0.1:8088/healthz
```
