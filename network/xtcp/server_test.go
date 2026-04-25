package xtcp_test

import (
	"net/http"
	_ "net/http/pprof"
	"testing"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/network/xtcp"
	"github.com/xbaseio/xbase/packet"
)

func TestServer_Simple(t *testing.T) {
	server := xtcp.NewServer()

	server.OnStart(func() {
		log.Info("server is started")
	})

	server.OnStop(func() {
		log.Info("server is stopped")
	})

	server.OnConnect(func(conn network.Conn) {
		log.Infof("connection is opened, connection id: %d", conn.ID())
	})

	server.OnDisconnect(func(conn network.Conn) {
		log.Infof("connection is closed, connection id: %d", conn.ID())
	})

	server.OnReceive(func(conn network.Conn, data []byte) {
		message, err := packet.UnpackMessage(data)
		if err != nil {
			log.Errorf("unpack message failed: %v", err)
			return
		}

		log.Infof("receive message from client, cid: %d, seq: %d, node id: %d, msg: %s", conn.ID(), message.Seq, message.NodeID, string(message.Buffer))

		msg, err := packet.PackMessage(&packet.Message{
			Seq:       1,
			NodeID:    1,
			MessageID: 1001,
			Buffer:    []byte("I'm fine~~"),
		})
		if err != nil {
			log.Errorf("pack message failed: %v", err)
			return
		}

		if err = conn.Push(msg); err != nil {
			log.Errorf("push message failed: %v", err)
		}
	})

	if err := server.Start(); err != nil {
		log.Fatalf("start server failed: %v", err)
	}

	select {}
}

func TestServer_Benchmark(t *testing.T) {
	server := xtcp.NewServer(
		xtcp.WithServerHeartbeatInterval(0),
	)

	server.OnStart(func() {
		log.Info("server is started")
	})

	server.OnReceive(func(conn network.Conn, data []byte) {
		message, err := packet.UnpackMessage(data)
		if err != nil {
			log.Errorf("unpack message failed: %v", err)
			return
		}

		msg, err := packet.PackMessage(&packet.Message{
			Seq:       message.Seq,
			NodeID:    message.NodeID,
			MessageID: message.MessageID,
			Buffer:    message.Buffer,
		})
		if err != nil {
			log.Errorf("pack message failed: %v", err)
			return
		}

		if err = conn.Push(msg); err != nil {
			log.Errorf("push message failed: %v", err)
			return
		}
	})

	if err := server.Start(); err != nil {
		log.Fatalf("start server failed: %v", err)
	}

	go func() {
		err := http.ListenAndServe(":8089", nil)
		if err != nil {
			log.Errorf("pprof server start failed: %v", err)
		}
	}()

	select {}
}
