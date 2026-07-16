package xsnowflake_test

import (
	"fmt"

	"github.com/xbaseio/xbase/utils/xsnowflake"
)

func ExampleGenerator() {
	// 每个同时运行的服务实例都必须使用不同的节点 ID。
	generator, err := xsnowflake.New(1)
	if err != nil {
		panic(err)
	}

	id, err := generator.Generate()
	if err != nil {
		panic(err)
	}

	info := xsnowflake.Parse(id)
	fmt.Printf("id is positive: %t\n", id > 0)
	fmt.Printf("node: %d\n", info.NodeID)

	// Output:
	// id is positive: true
	// node: 1
}
