package etc_test

import (
	"testing"

	"github.com/xbaseio/xbase/etc"
)

func Test_Get(t *testing.T) {
	v := etc.Get("c.redis.addrs.1A", "192.168.0.1:3308").String()
	t.Log(v)
}
