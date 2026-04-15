package encoding

import (
	"github.com/xbaseio/xbase/encoding/json"
	"github.com/xbaseio/xbase/encoding/msgpack"
	"github.com/xbaseio/xbase/encoding/proto"
	"github.com/xbaseio/xbase/encoding/toml"
	"github.com/xbaseio/xbase/encoding/xml"
	"github.com/xbaseio/xbase/encoding/yaml"
	"github.com/xbaseio/xbase/log"
)

var codecs = make(map[string]Codec)

func init() {
	Register(json.DefaultCodec)
	Register(proto.DefaultCodec)
	Register(toml.DefaultCodec)
	Register(xml.DefaultCodec)
	Register(yaml.DefaultCodec)
	Register(msgpack.DefaultCodec)
}

type Codec interface {
	// Name 编解码器类型
	Name() string
	// Marshal 编码
	Marshal(v any) ([]byte, error)
	// Unmarshal 解码
	Unmarshal(data []byte, v any) error
}

// Register 注册编解码器
func Register(codec Codec) {
	if codec == nil {
		log.Fatal("can't register a invalid codec")
	}

	name := codec.Name()

	if name == "" {
		log.Fatal("can't register a codec without name")
	}

	if _, ok := codecs[name]; ok {
		log.Warnf("the old %s codec will be overwritten", name)
	}

	codecs[name] = codec
}

// Invoke 调用编解码器
func Invoke(name string) Codec {
	codec, ok := codecs[name]
	if !ok {
		log.Fatalf("%s codec is not registered", name)
	}

	return codec
}
