package gate

import (
	"testing"

	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/utils/xjwt"
)

func TestParseLoginToken(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{
			name: "raw token",
			body: []byte("abc.def.ghi"),
			want: "abc.def.ghi",
		},
		{
			name: "bearer prefix",
			body: []byte("Bearer abc.def.ghi"),
			want: "abc.def.ghi",
		},
		{
			name: "json token",
			body: []byte(`{"token":"abc.def.ghi"}`),
			want: "abc.def.ghi",
		},
		{
			name: "json bearer token",
			body: []byte(`{"token":"Bearer abc.def.ghi"}`),
			want: "abc.def.ghi",
		},
		{
			name: "empty",
			body: []byte("   "),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLoginToken(tt.body); got != tt.want {
				t.Fatalf("parseLoginToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProxyIsLoginMessage(t *testing.T) {
	jwt, err := initTestJWT()
	if err != nil {
		t.Fatal(err)
	}

	p := &proxy{gate: &Gate{opts: &options{
		jwt:            jwt,
		loginMessageID: 1000,
		lobbyGameID:    0,
	}}}

	msg := &packet.Message{GameID: 0, MessageID: 1000}
	if !p.isLoginMessage(msg) {
		t.Fatal("expected login message")
	}

	msg.GameID = 1
	if p.isLoginMessage(msg) {
		t.Fatal("unexpected login message for other game")
	}
}

func initTestJWT() (*xjwt.JWT, error) {
	return xjwt.NewJWT(
		xjwt.WithSecretKey("test-secret"),
		xjwt.WithIdentityKey("uid"),
	)
}
