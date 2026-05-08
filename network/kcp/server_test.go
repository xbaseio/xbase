package kcp_test

import (
	"net/http"
	_ "net/http/pprof"
	"testing"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/network/kcp"
	"github.com/xbaseio/xbase/packet"
)

func TestServer_Simple(t *testing.T) {
	server := kcp.NewServer()

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
		message, _, err := packet.UnpackMessage(data)
		if err != nil {
			log.Errorf("unpack message failed: %v", err)
			return
		}

		log.Infof("receive message from client, cid: %d, seq: %d, game id: %d, msg: %s", conn.ID(), message.Seq, message.GameID, string(message.Buffer))

		msg, err := packet.PackMessage(&packet.Message{
			Seq:       1,
			GameID:    1,
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
	server := kcp.NewServer()

	server.OnStart(func() {
		log.Info("server is started")
	})

	server.OnReceive(func(conn network.Conn, data []byte) {
		message, _, err := packet.UnpackMessage(data)
		if err != nil {
			log.Errorf("unpack message failed: %v", err)
			return
		}

		msg, err := packet.PackMessage(&packet.Message{
			Seq:       message.Seq,
			GameID:    message.GameID,
			MessageID: message.MessageID,
			Buffer:    message.Buffer,
		})
		if err != nil {
			log.Errorf("pack message failed: %v", err)
			return
		}

		if err = conn.Send(msg); err != nil {
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
