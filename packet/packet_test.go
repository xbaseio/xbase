package packet_test

import (
	"bytes"
	"testing"

	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/utils/xrand"
)

var packer = packet.NewPacker()

func TestDefaultPacker_ReadMessage(t *testing.T) {
	data, err := packer.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte("hello world"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(data)

	reader := bytes.NewReader(data)

	message, err := packer.ReadMessage(reader)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(message)
}

func TestDefaultPacker_PackBuffer(t *testing.T) {
	buf, err := packer.PackBuffer(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte("hello world"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(buf.Bytes())

	message, _, err := packer.UnpackMessage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	buf.Release()

	t.Logf("seq: %d", message.Seq)
	t.Logf("node id: %d", message.GameID)
	t.Logf("message id: %d", message.MessageID)
	t.Logf("buffer: %s", string(message.Buffer))
}

func TestDefaultPacker_UnpackMessage_HalfPacket(t *testing.T) {
	buf, err := packer.PackBuffer(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte("hello world"),
	})
	if err != nil {
		t.Fatal(err)
	}

	raw := buf.Bytes()
	buf.Release()

	if msg, _, err := packer.UnpackMessage(raw[:10]); err != nil {
		t.Fatal(err)
	} else if msg != nil {
		t.Fatal("expected half packet")
	}
}

func TestDefaultPacker_ReadMessage_UnpackMessage(t *testing.T) {
	data, err := packer.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte("hello world"),
	})
	if err != nil {
		t.Fatal(err)
	}

	reader := bytes.NewReader(data)

	raw, err := packer.ReadMessage(reader)
	if err != nil {
		t.Fatal(err)
	}

	message, _, err := packer.UnpackMessage(raw)
	if err != nil {
		t.Fatal(err)
	}

	if message == nil {
		t.Fatal("expected message")
	}

	if string(message.Buffer) != "hello world" {
		t.Fatalf("unexpected body: %s", message.Buffer)
	}
}

func TestDefaultPacker_PackBuffer_NotWireFormat(t *testing.T) {
	buf, err := packer.PackBuffer(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte("hello world"),
	})
	if err != nil {
		t.Fatal(err)
	}

	raw := buf.Bytes()
	buf.Release()

	reader := bytes.NewReader(raw)
	if _, err := packer.ReadMessage(reader); err == nil {
		t.Fatal("PackBuffer must not be parsed by ReadMessage")
	}
}

func TestDefaultPacker_PackMessage_EmptyBody(t *testing.T) {
	data, err := packer.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
	})
	if err != nil {
		t.Fatal(err)
	}

	message, _, err := packer.UnpackMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	if message == nil {
		t.Fatal("expected message")
	}

	if len(message.Buffer) != 0 {
		t.Fatalf("expected empty body, got %q", message.Buffer)
	}
}

func TestDefaultPacker_PackMessage(t *testing.T) {
	data, err := packer.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte("hello world"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(data)

	message, _, err := packer.UnpackMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("seq: %d", message.Seq)
	t.Logf("game id: %d", message.GameID)
	t.Logf("message id: %d", message.MessageID)
	t.Logf("buffer: %s", string(message.Buffer))
}

func BenchmarkDefaultPacker_ReadBuffer(b *testing.B) {
	data, err := packer.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte(xrand.Letters(2048)),
	})
	if err != nil {
		b.Fatal(err)
	}

	reader := bytes.NewReader(data)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		if buf, err := packer.ReadBuffer(reader); err != nil {
			b.Fatal(err)
		} else {
			buf.Release()
		}

		reader.Reset(data)
	}
}

func BenchmarkDefaultPacker_ReadMessage(b *testing.B) {
	data, err := packer.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte(xrand.Letters(2048)),
	})
	if err != nil {
		b.Fatal(err)
	}

	reader := bytes.NewReader(data)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		if _, err = packer.ReadMessage(reader); err != nil {
			b.Fatal(err)
		}

		reader.Reset(data)
	}
}

func BenchmarkDefaultPacker_PackBuffer(b *testing.B) {
	buffer := []byte(xrand.Letters(1024))

	b.ResetTimer()
	b.SetBytes(int64(len(buffer)))

	for i := 0; i < b.N; i++ {
		buf, err := packer.PackBuffer(&packet.Message{
			Seq:       1,
			GameID:    1,
			MessageID: 1001,
			Buffer:    buffer,
		})
		if err != nil {
			b.Fatal(err)
		}

		buf.Release()
	}
}

func BenchmarkDefaultPacker_PackMessage(b *testing.B) {
	buffer := []byte(xrand.Letters(1024))

	b.ResetTimer()
	b.SetBytes(int64(len(buffer)))

	for i := 0; i < b.N; i++ {
		_, err := packer.PackMessage(&packet.Message{
			Seq:       1,
			GameID:    1,
			MessageID: 1001,
			Buffer:    buffer,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDefaultPacker_UnpackMessage(b *testing.B) {
	buf, err := packer.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte(xrand.Letters(1024)),
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(buf)))

	for i := 0; i < b.N; i++ {
		if _, _, err := packer.UnpackMessage(buf); err != nil {
			b.Fatal(err)
		}
	}
}
