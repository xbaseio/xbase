package main

import (
	"github.com/xbaseio/xbase"
	xhttp "github.com/xbaseio/xbase/component/http"
	"github.com/xbaseio/xbase/component/sse"
)

func main() {
	httpSrv := xhttp.NewServer(
		xhttp.WithAddr(":8088"),
	)

	sseSrv := sse.NewServer(
		sse.WithPath("/events"),
	)

	sseSrv.MountHTTP(httpSrv.Proxy())

	httpSrv.Proxy().Router().Get("/publish", func(ctx xhttp.Context) error {
		topic := ctx.Query("topic", "lobby")
		text := ctx.Query("text", "hello")

		delivered := sseSrv.Proxy().Publish(topic, sse.Event{
			Event: "message",
			Data: map[string]any{
				"topic": topic,
				"text":  text,
			},
		})

		return ctx.Success(map[string]any{
			"topic":     topic,
			"text":      text,
			"delivered": delivered,
		})
	})

	httpSrv.Proxy().Router().Get("/broadcast", func(ctx xhttp.Context) error {
		text := ctx.Query("text", "broadcast")

		delivered := sseSrv.Proxy().Broadcast(sse.Event{
			Event: "broadcast",
			Data: map[string]any{
				"text": text,
			},
		})

		return ctx.Success(map[string]any{
			"text":      text,
			"delivered": delivered,
		})
	})

	container := xbase.NewContainer()
	container.Add(httpSrv)
	container.Serve()
}
