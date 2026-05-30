package gate

import "github.com/xbaseio/xbase/network"

type inboundMessage struct {
	conn network.Conn
	data []byte
}
