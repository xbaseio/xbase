package node

import (
	"context"
	"time"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/xerrors"
)

type EventHandler func(ctx Context)

type Trigger struct {
	node    *Node
	events  map[cluster.Event]EventHandler
	evtChan chan *event
}

func newTrigger(node *Node) *Trigger {
	return &Trigger{
		node:    node,
		events:  make(map[cluster.Event]EventHandler, 3),
		evtChan: make(chan *event, 4096),
	}
}

func (e *Trigger) trigger(kind cluster.Event, gid string, cid, uid int64) error {
	evt := e.node.evtPool.Get().(*event)
	evt.ctx = context.Background()
	evt.event = kind
	evt.gid = gid
	evt.cid = cid
	evt.uid = uid

	timeout := e.node.opts.deliverTimeout
	if timeout <= 0 {
		e.evtChan <- evt
		return nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case e.evtChan <- evt:
		return nil
	case <-timer.C:
		evt.reset()
		e.node.evtPool.Put(evt)
		log.Warnf("node event queue full, drop event: %v uid: %d", kind.String(), uid)
		return xerrors.ErrDeliverQueueFull
	}
}

func (e *Trigger) receive() <-chan *event {
	return e.evtChan
}

func (e *Trigger) close() {
	close(e.evtChan)
}

// 处理事件消息
func (e *Trigger) handle(evt *event) {
	version := evt.incrVersion()

	if handler, ok := e.events[evt.event]; ok {
		xcall.Call(func() { handler(evt) })

		evt.compareVersionExecDefer(version)
	}

	evt.compareVersionRecycle(version)
}

// AddEventHandler 添加事件处理器
func (e *Trigger) AddEventHandler(event cluster.Event, handler EventHandler) {
	if e.node.getState() != cluster.Shut {
		log.Warnf("the node server is working, can't add Event handler")
		return
	}

	e.events[event] = handler
}
