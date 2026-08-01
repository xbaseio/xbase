package redis

import (
	"maps"
	"testing"

	"github.com/xbaseio/xbase/locate"
)

func TestNodeBindingCodec(t *testing.T) {
	want := locate.NodeBinding{
		NID: "node-1",
		Metadata: map[string]string{
			"agentCode": "agent001",
			"roomID":    "10001",
		},
	}

	encoded, err := marshalNodeBinding(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := unmarshalNodeBinding(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got.NID != want.NID || !maps.Equal(got.Metadata, want.Metadata) {
		t.Fatalf("binding = %#v, want %#v", got, want)
	}
}

func TestNodeBindingCodecReadsLegacyNodeID(t *testing.T) {
	binding, err := unmarshalNodeBinding("legacy-node")
	if err != nil {
		t.Fatal(err)
	}
	if binding.NID != "legacy-node" || len(binding.Metadata) != 0 {
		t.Fatalf("binding = %#v", binding)
	}
}
