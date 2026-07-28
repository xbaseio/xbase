package crypto

import (
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

type Encryptor interface {
	// Name 名称
	Name() string
	// Encrypt 加密
	Encrypt(data []byte) ([]byte, error)
	// Decrypt 解密
	Decrypt(data []byte) ([]byte, error)
}

var encryptors = make(map[string]Encryptor)

// RegisterEncryptor 注册加密器
func RegisterEncryptor(encryptor Encryptor) {
	if encryptor == nil {
		xlog.Logger().Fatal("can't register a invalid encryptor")
	}

	name := encryptor.Name()

	if name == "" {
		xlog.Logger().Fatal("can't register a encryptor without name")
	}

	if _, ok := encryptors[name]; ok {
		xlog.Logger().Warn("the old encryptor will be overwritten", zap.Any("name", name))
	}

	encryptors[name] = encryptor
}

// InvokeEncryptor 调用加密器
func InvokeEncryptor(name string) Encryptor {
	encryptor, ok := encryptors[name]
	if !ok {
		xlog.Logger().Fatal("encryptor is not registered", zap.Any("name", name))
	}

	return encryptor
}
