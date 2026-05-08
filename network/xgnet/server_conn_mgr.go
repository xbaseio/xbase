package xgnet

import (
	"sync"
	"sync/atomic"

	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/utils/xnet"
	"github.com/xbaseio/xbase/xerrors"
)

const serverConnPartitionCount = 100

type serverConnMgr struct {
	id         atomic.Int64
	total      atomic.Int64
	server     *server
	pool       sync.Pool
	partitions []*partition
}

func newServerConnMgr(server *server) *serverConnMgr {
	cm := &serverConnMgr{
		server:     server,
		partitions: make([]*partition, serverConnPartitionCount),
	}

	cm.pool = sync.Pool{
		New: func() any {
			return &gnetConn{}
		},
	}

	for i := 0; i < len(cm.partitions); i++ {
		cm.partitions[i] = &partition{
			connections: make(map[int64]*gnetConn),
		}
	}

	return cm
}

// 分配连接
func (cm *serverConnMgr) allocate(c xnet.Conn) (*gnetConn, error) {
	if !cm.tryAddTotal() {
		return nil, xerrors.ErrTooManyConnection
	}

	id := cm.id.Add(1)

	conn := cm.pool.Get().(*gnetConn)
	conn.init(cm, id, c)

	cm.partition(id).store(id, conn)

	return conn, nil
}

func (cm *serverConnMgr) tryAddTotal() bool {
	maxConnNum := int64(cm.server.opts.maxConnNum)

	for {
		old := cm.total.Load()

		if maxConnNum > 0 && old >= maxConnNum {
			return false
		}

		if cm.total.CompareAndSwap(old, old+1) {
			return true
		}
	}
}

// 回收连接
func (cm *serverConnMgr) recycle(id int64) {
	p := cm.partition(id)

	conn, ok := p.delete(id)
	if !ok || conn == nil {
		return
	}

	conn.reset()
	cm.pool.Put(conn)
	cm.total.Add(-1)
}

// 关闭所有连接
func (cm *serverConnMgr) close() {
	var wg sync.WaitGroup

	wg.Add(len(cm.partitions))

	for i := range cm.partitions {
		p := cm.partitions[i]

		xcall.Go(func() {
			p.close()
			wg.Done()
		})
	}

	wg.Wait()
}

func (cm *serverConnMgr) partition(id int64) *partition {
	index := int(id % int64(len(cm.partitions)))
	if index < 0 {
		index = -index
	}

	return cm.partitions[index]
}

type partition struct {
	rw          sync.RWMutex
	connections map[int64]*gnetConn
}

// 存储连接
func (p *partition) store(id int64, conn *gnetConn) {
	p.rw.Lock()
	p.connections[id] = conn
	p.rw.Unlock()
}

// 删除连接
func (p *partition) delete(id int64) (*gnetConn, bool) {
	p.rw.Lock()
	conn, ok := p.connections[id]
	if ok {
		delete(p.connections, id)
	}
	p.rw.Unlock()

	return conn, ok
}

// 关闭该分片内所有连接
func (p *partition) close() {
	// 先拷贝，不能直接 range map 后 Close。
	// Close 过程中会触发 recycle/delete，直接遍历 map 会有并发修改风险。
	p.rw.RLock()
	list := make([]*gnetConn, 0, len(p.connections))
	for _, conn := range p.connections {
		if conn != nil {
			list = append(list, conn)
		}
	}
	p.rw.RUnlock()

	for _, conn := range list {
		_ = conn.Close(true)
	}
}
