package kcp_test

import (
	"net/http"
	_ "net/http/pprof"
	"testing"

	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/network/kcp"
	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

func TestServer_Simple(t *testing.T) {
	server := kcp.NewServer()

	server.OnStart(func() {
		xlog.Logger().Info("server is started")
	})

	server.OnStop(func() {
		xlog.Logger().Info("server is stopped")
	})

	server.OnConnect(func(conn network.Conn) {
		xlog.Logger().Info("connection is opened, connection id", zap.Any("iD", conn.ID()))
	})

	server.OnDisconnect(func(conn network.Conn) {
		xlog.Logger().Info("connection is closed, connection id", zap.Any("iD", conn.ID()))
	})

	server.OnReceive(func(conn network.Conn, data []byte) {
		message, _, err := packet.UnpackMessage(data)
		if err != nil {
			xlog.Logger().Error("unpack message failed", zap.Error(err))
			return
		}
		xlog.Logger().Info("receive message from client, cid: , seq: , game id: , msg", zap.Any("iD", conn.ID()), zap.Any("seq", message.Seq), zap.Any("gameID", message.GameID), zap.String("buffer", string(message.Buffer)))

		msg, err := packet.PackMessage(&packet.Message{
			Seq:       1,
			GameID:    1,
			MessageID: 1001,
			Buffer:    []byte("I'm fine~~"),
		})
		if err != nil {
			xlog.Logger().Error("pack message failed", zap.Error(err))
			return
		}

		if err = conn.Push(msg); err != nil {
			xlog.Logger().Error("push message failed", zap.Error(err))
		}
	})

	if err := server.Start(); err != nil {
		xlog.Logger().Fatal("start server failed", zap.Error(err))
	}

	select {}
}

func TestServer_Benchmark(t *testing.T) {
	server := kcp.NewServer()

	server.OnStart(func() {
		xlog.Logger().Info("server is started")
	})

	server.OnReceive(func(conn network.Conn, data []byte) {
		message, _, err := packet.UnpackMessage(data)
		if err != nil {
			xlog.Logger().Error("unpack message failed", zap.Error(err))
			return
		}

		msg, err := packet.PackMessage(&packet.Message{
			Seq:       message.Seq,
			GameID:    message.GameID,
			MessageID: message.MessageID,
			Buffer:    message.Buffer,
		})
		if err != nil {
			xlog.Logger().Error("pack message failed", zap.Error(err))
			return
		}

		if err = conn.Send(msg); err != nil {
			xlog.Logger().Error("push message failed", zap.Error(err))
			return
		}
	})

	if err := server.Start(); err != nil {
		xlog.Logger().Fatal("start server failed", zap.Error(err))
	}

	go func() {
		err := http.ListenAndServe(":8089", nil)
		if err != nil {
			xlog.Logger().Error("pprof server start failed", zap.Error(err))
		}
	}()

	select {}
}
