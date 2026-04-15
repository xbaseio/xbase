package limiter_test

import (
	"fmt"
	"testing"

	"github.com/xbaseio/xbase/core/limiter"
)

func TestLimiter_Allow(t *testing.T) {
	l := limiter.NewLimiter(10, 1)

	for i := 0; i < 15; i++ {
		if l.Allow() {
			fmt.Println("请求允许", i+1)
		} else {
			fmt.Println("请求被限流", i+1)
		}

		//time.Sleep(100 * time.Millisecond)
	}
}
