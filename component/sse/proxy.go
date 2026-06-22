package sse

type Proxy struct {
	server *Server
}

func newProxy(s *Server) *Proxy {
	return &Proxy{server: s}
}

func (p *Proxy) Publish(topic string, event Event) int {
	return p.server.broker.publish(topic, event)
}

func (p *Proxy) Broadcast(event Event) int {
	return p.server.broker.publish("", event)
}

func (p *Proxy) ClientCount() int {
	return p.server.broker.count()
}
