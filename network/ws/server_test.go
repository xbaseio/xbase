package ws_test

import (
	"net/http"
	"testing"

	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/network/ws"
	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

func TestServer(t *testing.T) {
	server := ws.NewServer()
	server.OnStart(func() {
		t.Logf("server is started")
	})
	server.OnConnect(func(conn network.Conn) {
		t.Logf("connection is opened, connection id: %d", conn.ID())
	})
	server.OnDisconnect(func(conn network.Conn) {
		t.Logf("connection is closed, connection id: %d", conn.ID())
	})
	server.OnReceive(func(conn network.Conn, data []byte) {
		message, _, err := packet.UnpackMessage(data)
		if err != nil {
			t.Error(err)
			return
		}

		t.Logf("receive msg from client, connection id: %d, seq: %d, game id: %d, msg: %s", conn.ID(), message.Seq, message.GameID, string(message.Buffer))

		msg, err := packet.PackMessage(&packet.Message{
			Seq:       1,
			GameID:    1,
			MessageID: 1001,
			Buffer:    []byte("I'm fine~~"),
		})
		if err != nil {
			t.Fatal(err)
		}

		if err = conn.Push(msg); err != nil {
			t.Error(err)
		}
	})
	server.OnUpgrade(func(w http.ResponseWriter, r *http.Request) (allowed bool) {
		return true
	})

	if err := server.Start(); err != nil {
		t.Fatal(err)
	}

	xcall.Go(func() {
		err := http.ListenAndServe(":8089", nil)
		if err != nil {
			xlog.Logger().Error("pprof server start failed", zap.Error(err))
		}
	})

	select {}
}

func TestServer_Benchmark(t *testing.T) {
	server := ws.NewServer()
	server.OnStart(func() {
		t.Logf("server is started")
	})
	server.OnReceive(func(conn network.Conn, data []byte) {
		if _, _, err := packet.UnpackMessage(data); err != nil {
			t.Error(err)
			return
		}

		msg, err := packet.PackMessage(&packet.Message{
			Seq:       1,
			GameID:    101,
			MessageID: 1001,
			Buffer:    []byte("I'm fine~~"),
		})
		if err != nil {
			t.Fatal(err)
		}

		if err = conn.Push(msg); err != nil {
			t.Error(err)
		}
	})
	server.OnUpgrade(func(w http.ResponseWriter, r *http.Request) (allowed bool) {
		return true
	})

	if err := server.Start(); err != nil {
		t.Fatal(err)
	}

	xcall.Go(func() {
		err := http.ListenAndServe(":8089", nil)
		if err != nil {
			xlog.Logger().Error("pprof server start failed", zap.Error(err))
		}
	})

	select {}
}
