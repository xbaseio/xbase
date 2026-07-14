package sse

import (
	"bufio"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	xhttp "github.com/xbaseio/xbase/component/http"
	"github.com/xbaseio/xbase/utils/xuuid"
)

func (s *Server) MountHTTP(proxy *xhttp.Proxy, path ...string) {
	if proxy == nil {
		return
	}

	eventsPath := s.opts.path
	if len(path) > 0 && path[0] != "" {
		eventsPath = path[0]
	}

	router := proxy.Router()
	router.Get(s.opts.healthPath, func(ctx xhttp.Context) error {
		return s.fiberHealthHandler(ctx.CTX())
	})
	router.Get(eventsPath, func(ctx xhttp.Context) error {
		return s.fiberStreamHandler(ctx.CTX())
	})
}

func (s *Server) FiberHandler() fiber.Handler {
	return s.fiberStreamHandler
}

func (s *Server) FiberHealthHandler() fiber.Handler {
	return s.fiberHealthHandler
}

func (s *Server) fiberHealthHandler(c fiber.Ctx) error {
	c.Set("Content-Type", "application/json; charset=utf-8")
	return c.SendString(`{"ok":true,"clients":` + strconv.Itoa(s.broker.count()) + `}`)
}

func (s *Server) fiberStreamHandler(c fiber.Ctx) error {
	clientID, topics, err := s.resolveFiberClient(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	client := s.newClient(clientID, topics)
	s.broker.add(client)
	defer s.broker.remove(client.id)

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	streamCtx := c.Context()
	return c.SendStreamWriter(func(w *bufio.Writer) {
		_, _ = w.Write(Event{
			ID:    s.nextID(),
			Event: "connected",
			Data: map[string]any{
				"clientID": client.id,
				"topics":   topics,
			},
		}.Bytes())
		_ = w.Flush()

		heartbeat := time.NewTicker(s.opts.heartbeatInterval)
		defer heartbeat.Stop()

		for {
			select {
			case <-streamCtx.Done():
				return
			case <-heartbeat.C:
				_, _ = w.Write([]byte(": ping\n\n"))
				if err := w.Flush(); err != nil {
					return
				}
			case event := <-client.send:
				if event.ID == "" {
					event.ID = s.nextID()
				}
				_, _ = w.Write(event.Bytes())
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
}

func (s *Server) resolveFiberClient(c fiber.Ctx) (string, []string, error) {
	if s.opts.connectHandler != nil {
		req, err := http.NewRequest(c.Method(), c.OriginalURL(), nil)
		if err != nil {
			return "", nil, err
		}
		req.Header = c.GetReqHeaders()
		return s.opts.connectHandler(req)
	}

	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = xuuid.UUID()
	}

	topics := collectTopicsFromValues(parseFiberTopics(c.Request().URI().QueryString(), s.opts.topicQueryKey))
	return clientID, topics, nil
}

func parseFiberTopics(raw []byte, key string) []string {
	if len(raw) == 0 {
		return nil
	}

	values, err := url.ParseQuery(string(raw))
	if err != nil {
		return nil
	}
	return values[key]
}

func collectTopicsFromValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	topics := make([]string, 0, len(values))
	for _, value := range values {
		for topic := range strings.SplitSeq(value, ",") {
			topic = strings.TrimSpace(topic)
			if topic != "" {
				topics = append(topics, topic)
			}
		}
	}

	return topics
}
