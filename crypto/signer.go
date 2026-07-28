package crypto

import (
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

type Signer interface {
	// Name 名称
	Name() string
	// Sign 签名
	Sign(data []byte) ([]byte, error)
	// Verify 验签
	Verify(data []byte, signature []byte) (bool, error)
}

var signers = make(map[string]Signer)

// RegisterSigner 注册签名器
func RegisterSigner(signer Signer) {
	if signer == nil {
		xlog.Logger().Fatal("can't register a invalid signer")
	}

	name := signer.Name()

	if name == "" {
		xlog.Logger().Fatal("can't register a signer without name")
	}

	if _, ok := signers[name]; ok {
		xlog.Logger().Warn("the old signer will be overwritten", zap.Any("name", name))
	}

	signers[name] = signer
}

// InvokeSigner 调用签名器
func InvokeSigner(name string) Signer {
	signer, ok := signers[name]
	if !ok {
		xlog.Logger().Fatal("signer is not registered", zap.Any("name", name))
	}

	return signer
}
