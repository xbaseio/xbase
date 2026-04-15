//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd) && !gc_opt

package xnet

import (
	"sync/atomic"

	"github.com/xbaseio/xbase/utils/xgfd"
)

type connMatrix struct {
	connCount int32
	connMap   map[int]*conn
}

func (cm *connMatrix) init() {
	cm.connMap = make(map[int]*conn)
}

func (cm *connMatrix) iterate(f func(*conn) bool) {
	for _, c := range cm.connMap {
		if c != nil {
			if !f(c) {
				return
			}
		}
	}
}

func (cm *connMatrix) incCount(_ int, delta int32) {
	atomic.AddInt32(&cm.connCount, delta)
}

func (cm *connMatrix) loadCount() (n int32) {
	return atomic.LoadInt32(&cm.connCount)
}

func (cm *connMatrix) addConn(c *conn, index int) {
	c.xgfd = xgfd.NewGFD(c.fd, index, 0, 0)
	cm.connMap[c.fd] = c
	cm.incCount(0, 1)
}

func (cm *connMatrix) delConn(c *conn) {
	delete(cm.connMap, c.fd)
	cm.incCount(0, -1)
}

func (cm *connMatrix) getConn(fd int) *conn {
	return cm.connMap[fd]
}

/*
func (cm *connMatrix) getConnByGFD(fd xgfd.GFD) *conn {
	return cm.connMap[fd.Fd()]
}
*/
