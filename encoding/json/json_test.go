package json

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodecRoundTrip(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	want := payload{Name: "xbase", Count: 2}
	data, err := Marshal(want)
	require.NoError(t, err)

	var got payload
	require.NoError(t, Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestCodecUsesV2StrictDefaults(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	var got payload
	require.Error(t, Unmarshal([]byte(`{"name":"first","name":"second"}`), &got))

	got = payload{}
	require.NoError(t, Unmarshal([]byte(`{"Name":"ignored"}`), &got))
	require.Empty(t, got.Name)
}
