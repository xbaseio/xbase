package protocol_test

import (
	"testing"

	"github.com/xbaseio/xbase/internal/transporter/internal/protocol"
	"github.com/xbaseio/xbase/session"
)

func TestDecodeStatReq(t *testing.T) {
	buffer := protocol.EncodeStatReq(1, session.User)

	seq, kind, err := protocol.DecodeStatReq(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("seq: %v", seq)
	t.Logf("kind: %v", kind)
}

func TestDecodeStatRes(t *testing.T) {
	buffer := protocol.EncodeStatRes(1, 2000)

	code, total, err := protocol.DecodeStatRes(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("code: %v", code)
	t.Logf("total: %v", total)
}
