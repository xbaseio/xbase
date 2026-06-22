package sse

import (
	"testing"
	"time"
)

func TestEventBytes(t *testing.T) {
	buf := string(Event{
		ID:    "1",
		Event: "message",
		Retry: 3000,
		Data:  "hello\nworld",
	}.Bytes())

	want := "id: 1\nevent: message\nretry: 3000\ndata: hello\ndata: world\n\n"
	if buf != want {
		t.Fatalf("unexpected event bytes:\nwant=%q\ngot =%q", want, buf)
	}
}

func TestCollectTopicsFromValues(t *testing.T) {
	topics := collectTopicsFromValues([]string{"lobby,battle", "chat"})
	if len(topics) != 3 || topics[0] != "lobby" || topics[1] != "battle" || topics[2] != "chat" {
		t.Fatalf("unexpected topics: %#v", topics)
	}
}

func TestProxyPublish(t *testing.T) {
	s := NewServer(WithHeartbeatInterval(time.Second))
	c := &client{
		id:     "c1",
		topics: map[string]struct{}{"lobby": {}},
		send:   make(chan Event, 1),
	}
	s.broker.add(c)

	if n := s.Proxy().Publish("lobby", Event{Event: "message", Data: "ok"}); n != 1 {
		t.Fatalf("unexpected publish count: %d", n)
	}
}
