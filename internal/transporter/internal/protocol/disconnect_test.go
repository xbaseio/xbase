package protocol_test

import (
	"testing"

	"github.com/xbaseio/xbase/internal/transporter/internal/codes"
	"github.com/xbaseio/xbase/internal/transporter/internal/protocol"
	"github.com/xbaseio/xbase/session"
)

func TestEncodeDisconnectReq(t *testing.T) {
	buffer := protocol.EncodeDisconnectReq(1, session.User, 3, true)

	t.Log(buffer.Bytes())
}

func TestDecodeDisconnectReq(t *testing.T) {
	buffer := protocol.EncodeDisconnectReq(1, session.User, 3, false)

	seq, kind, target, force, err := protocol.DecodeDisconnectReq(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("seq: %v", seq)
	t.Logf("kind: %v", kind)
	t.Logf("target: %v", target)
	t.Logf("force: %v", force)
}

func TestEncodeDisconnectRes(t *testing.T) {
	buffer := protocol.EncodeDisconnectRes(1, codes.OK)

	t.Log(buffer.Bytes())
}

func TestDecodeDisconnectRes(t *testing.T) {
	buffer := protocol.EncodeDisconnectRes(1, codes.OK)

	code, err := protocol.DecodeDisconnectRes(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("code: %v", code)
}
