package protocol_test

import (
	"testing"

	"github.com/xbaseio/xbase/internal/transporter/internal/codes"
	"github.com/xbaseio/xbase/internal/transporter/internal/protocol"
	"github.com/xbaseio/xbase/session"
)

func TestDecodeIsOnlineReq(t *testing.T) {
	buffer := protocol.EncodeIsOnlineReq(1, session.User, 1)

	seq, kind, target, err := protocol.DecodeIsOnlineReq(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("seq: %v", seq)
	t.Logf("kind: %v", kind)
	t.Logf("target: %v", target)
}

func TestDecodeIsOnlineRes(t *testing.T) {
	buffer := protocol.EncodeIsOnlineRes(1, codes.NotFoundSession, false)

	code, isOnline, err := protocol.DecodeIsOnlineRes(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("code: %v", code)
	t.Logf("isOnline: %v", isOnline)
}
