package sse

import (
	"fmt"
	"sync/atomic"
)

type Server struct {
	opts   *options
	proxy  *Proxy
	broker *broker
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

func (s *Server) nextID() string {
	return fmt.Sprintf("%d", s.seq.Add(1))
}

func (s *Server) newClient(clientID string, topics []string) *client {
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
	return c
}
