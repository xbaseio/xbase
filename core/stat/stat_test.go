package stat_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/xbaseio/xbase/core/stat"
)

func TestStat(t *testing.T) {
	fi, err := stat.Stat("stat_linux.go")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(fi.CreateTime())
}

func Test_Watcher(t *testing.T) {
	cache := stat.NewCache(10 * time.Second)

	watcher, err := stat.NewWatcher(cache)
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	watcher.AddHandler(func(path string, op fsnotify.Op) {
		fmt.Println("file changed:", path, "op:", op)
	})

	if err := watcher.Watch("./"); err != nil {
		panic(err)
	}

	info, err := cache.Get("./stat_linux.go")
	if err != nil {
		panic(err)
	}

	fmt.Println("name:", info.Name())
	fmt.Println("mtime:", info.ModifyTime())

	select {}

}
