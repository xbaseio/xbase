package xnet

import (
	"hash/crc32"
	"net"
	"sync"
	"sync/atomic"

	"github.com/xbaseio/xbase/utils/xbs"
)

// LoadBalancing 表示负载均衡算法类型。
type LoadBalancing int

const (
	// RoundRobin 按轮询方式将新连接分配到事件循环。
	RoundRobin LoadBalancing = iota

	// LeastConnections 将新连接分配到当前活跃连接数最少的事件循环。
	LeastConnections

	// SourceAddrHash 根据远端地址哈希将新连接分配到事件循环。
	SourceAddrHash
)

type (
	// loadBalancer 定义事件循环负载均衡器接口。
	loadBalancer interface {
		register(*eventloop)
		next(net.Addr) *eventloop
		index(int) *eventloop
		iterate(func(int, *eventloop) bool)
		len() int
	}

	// baseLoadBalancer 提供基础事件循环管理能力。
	baseLoadBalancer struct {
		mu         sync.RWMutex
		eventLoops []*eventloop
		size       int
	}

	// roundRobinLoadBalancer 使用轮询算法。
	roundRobinLoadBalancer struct {
		baseLoadBalancer
		nextIndex atomic.Uint64
	}

	// leastConnectionsLoadBalancer 使用最少连接算法。
	leastConnectionsLoadBalancer struct {
		baseLoadBalancer
	}

	// sourceAddrHashLoadBalancer 使用源地址哈希算法。
	sourceAddrHashLoadBalancer struct {
		baseLoadBalancer
	}
)

//
// =========================
// baseLoadBalancer
// =========================
//

// register 向负载均衡器注册一个新的事件循环。
func (lb *baseLoadBalancer) register(el *eventloop) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	el.idx = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
}

// index 根据索引返回对应的事件循环。
func (lb *baseLoadBalancer) index(i int) *eventloop {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if i < 0 || i >= lb.size {
		return nil
	}
	return lb.eventLoops[i]
}

// iterate 遍历所有事件循环。
// 这里先拷贝快照，避免在回调执行期间长时间持有锁。
func (lb *baseLoadBalancer) iterate(f func(int, *eventloop) bool) {
	lb.mu.RLock()
	snapshot := make([]*eventloop, len(lb.eventLoops))
	copy(snapshot, lb.eventLoops)
	lb.mu.RUnlock()

	for i, el := range snapshot {
		if !f(i, el) {
			break
		}
	}
}

// len 返回事件循环数量。
func (lb *baseLoadBalancer) len() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.size
}

//
// =========================
// RoundRobin
// =========================
//

// next 根据轮询算法返回下一个事件循环。
func (lb *roundRobinLoadBalancer) next(_ net.Addr) *eventloop {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.size == 0 {
		return nil
	}

	idx := lb.nextIndex.Add(1) - 1
	return lb.eventLoops[idx%uint64(lb.size)]
}

//
// =========================
// LeastConnections
// =========================
//

// next 返回当前活跃连接数最少的事件循环。
func (lb *leastConnectionsLoadBalancer) next(_ net.Addr) (el *eventloop) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.size == 0 {
		return nil
	}

	el = lb.eventLoops[0]
	minN := el.countConn()

	for _, candidate := range lb.eventLoops[1:] {
		if n := candidate.countConn(); n < minN {
			minN = n
			el = candidate
		}
	}

	return
}

//
// =========================
// SourceAddrHash
// =========================
//

// hash 将字符串计算为稳定的非负哈希值。
func (*sourceAddrHashLoadBalancer) hash(s string) int {
	v := int(crc32.ChecksumIEEE(xbs.StringToBytes(s)))
	if v >= 0 {
		return v
	}
	return -v
}

// next 根据远端地址哈希值选择事件循环。
func (lb *sourceAddrHashLoadBalancer) next(addr net.Addr) *eventloop {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.size == 0 {
		return nil
	}

	hashCode := lb.hash(addr.String())
	return lb.eventLoops[hashCode%lb.size]
}
