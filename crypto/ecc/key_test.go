package ecc_test

import (
	"fmt"
	"testing"

	"github.com/xbaseio/xbase/crypto/ecc"
)

func TestGenerateKey(t *testing.T) {
	key, err := ecc.GenerateKey(ecc.P256)
	if err != nil {
		t.Fatal(err)
	}

	v, err := key.MarshalPublicKey()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(v))
}

func TestKey_SaveKeyPair(t *testing.T) {
	key, err := ecc.GenerateKey(ecc.P256)
	if err != nil {
		t.Fatal(err)
	}

	err = key.SaveKeyPair("./pem", "key.pem")
	if err != nil {
		t.Fatal(err)
	}
}
