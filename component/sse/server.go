package sse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/component"
	"github.com/xbaseio/xbase/core/info"
	xnet "github.com/xbaseio/xbase/core/net"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xuuid"
)

var _ component.Component = &Server{}

type Server struct {
	component.Base
	opts   *options
	proxy  *Proxy
	broker *broker
	server *http.Server
	seq    atomic.Uint64
}

func NewServer(opts ...Option) *Server {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	s := &Server{
		opts:   o,
		broker: newBroker(),
	}
	s.proxy = newProxy(s)
	return s
}

func (s *Server) Name() string {
	return s.opts.name
}

func (s *Server) Proxy() *Proxy {
	return s.proxy
}

func (s *Server) Start() {
	listenAddr, exposeAddr, err := xnet.ParseAddr(s.opts.addr)
	if err != nil {
		log.Fatalf("sse addr parse failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(s.opts.healthPath, s.handleHealth)
	mux.HandleFunc(s.opts.path, s.handleStream)

	s.server = &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("sse server startup failed: %v", err)
		}
	}()

	info.PrintBoxInfo("SSE",
		fmt.Sprintf("Name: %s", s.Name()),
		fmt.Sprintf("Url: http://%s%s", exposeAddr, s.opts.path),
		fmt.Sprintf("Health: http://%s%s", exposeAddr, s.opts.healthPath),
	)
}

func (s *Server) Close() {
	if s.server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.opts.shutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		log.Warnf("sse server shutdown failed: %v", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"ok":true,"clients":%d}`, s.broker.count())))
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	clientID, topics, err := s.resolveClient(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c := &client{
		id:     clientID,
		topics: make(map[string]struct{}, len(topics)),
		send:   make(chan Event, s.opts.clientBuffer),
	}
	for _, topic := range topics {
		if topic != "" {
			c.topics[topic] = struct{}{}
		}
	}

	s.broker.add(c)
	defer s.broker.remove(c.id)

	headers := w.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")

	_, _ = w.Write(Event{
		ID:    s.nextID(),
		Event: "connected",
		Data: map[string]any{
			"clientID": c.id,
			"topics":   topics,
		},
	}.Bytes())
	flusher.Flush()

	heartbeat := time.NewTicker(s.opts.heartbeatInterval)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		case event := <-c.send:
			if event.ID == "" {
				event.ID = s.nextID()
			}
			_, _ = w.Write(event.Bytes())
			flusher.Flush()
		}
	}
}

func (s *Server) resolveClient(r *http.Request) (string, []string, error) {
	if s.opts.connectHandler != nil {
		return s.opts.connectHandler(r)
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = xuuid.UUID()
	}

	topics := collectTopics(r, s.opts.topicQueryKey)
	return clientID, topics, nil
}

func collectTopics(r *http.Request, key string) []string {
	values := r.URL.Query()[key]
	if len(values) == 0 {
		return nil
	}

	topics := make([]string, 0, len(values))
	for _, value := range values {
		for _, topic := range strings.Split(value, ",") {
			topic = strings.TrimSpace(topic)
			if topic != "" {
				topics = append(topics, topic)
			}
		}
	}
	return topics
}

func (s *Server) nextID() string {
	return fmt.Sprintf("%d", s.seq.Add(1))
}
