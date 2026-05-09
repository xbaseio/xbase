package protocol_test

import (
	"testing"

	"github.com/xbaseio/xbase/internal/transporter/internal/protocol"
)

func TestEncodeUnbindReq(t *testing.T) {
	buffer := protocol.EncodeUnbindReq(1, 2, 3)

	t.Log(buffer.Bytes())
}

func TestDecodeUnbindReq(t *testing.T) {
	buffer := protocol.EncodeUnbindReq(1, 214, 314)

	seq, uid, cid, err := protocol.DecodeUnbindReq(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("seq: %v", seq)
	t.Logf("uid: %v", uid)
	t.Logf("cid: %v", cid)
}

func TestEncodeUnbindRes(t *testing.T) {
	buffer := protocol.EncodeUnbindRes(1, 2)

	t.Log(buffer.Bytes())
}

func TestDecodeUnbindRes(t *testing.T) {
	buffer := protocol.EncodeUnbindRes(1, 2)

	code, err := protocol.DecodeUnbindRes(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("code: %v", code)
}
