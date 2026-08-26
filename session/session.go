package session

import (
	"net"
	"strings"
	"sync"

	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/xerrors"
)

const (
	Conn Kind = iota + 1 // 连接SESSION
	User                 // 用户SESSION
)

const channelAttrPrefix = "__session_channel__:"

type Kind int

func (k Kind) String() string {
	switch k {
	case Conn:
		return "conn"
	case User:
		return "user"
	default:
		return "unknown"
	}
}

type Session struct {
	rw       sync.RWMutex
	conns    map[int64]network.Conn               // 连接ID -> Conn
	users    map[int64]network.Conn               // 用户ID -> Conn
	channels map[string]map[network.Conn]struct{} // 频道名 -> Conn集合
}

func NewSession() *Session {
	return &Session{
		conns:    make(map[int64]network.Conn),
		users:    make(map[int64]network.Conn),
		channels: make(map[string]map[network.Conn]struct{}),
	}
}

// AddConn 添加连接
func (s *Session) AddConn(conn network.Conn) {
	if conn == nil {
		return
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	cid := conn.ID()
	uid := conn.UID()

	s.conns[cid] = conn

	if uid != 0 {
		// 同一个 uid 已经有旧连接时，先解除旧连接绑定，避免 users[uid] 指向混乱
		if oldConn, ok := s.users[uid]; ok && oldConn.ID() != cid {
			oldConn.Unbind()
		}

		s.users[uid] = conn
	}
}

// RemConn 移除连接
func (s *Session) RemConn(conn network.Conn) {
	if conn == nil {
		return
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	cid := conn.ID()
	uid := conn.UID()

	delete(s.conns, cid)

	// 只能删除当前连接对应的 uid 映射。
	// 防止旧连接断开时，把已经重连的新连接 users[uid] 删除掉。
	if uid != 0 {
		if curConn, ok := s.users[uid]; ok && curConn.ID() == cid {
			delete(s.users, uid)
		}
	}

	// 清理频道订阅
	conn.Attr().Visit(func(key, _ any) bool {
		attrKey, ok := key.(string)
		if !ok {
			return true
		}

		if !strings.HasPrefix(attrKey, channelAttrPrefix) {
			return true
		}

		channel := strings.TrimPrefix(attrKey, channelAttrPrefix)
		if channel != "" {
			s.doUnsubscribeLocked(channel, conn)
		}

		return true
	})
}

// Has 是否存在会话
func (s *Session) Has(kind Kind, target int64) (bool, error) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	switch kind {
	case Conn:
		_, ok := s.conns[target]
		return ok, nil
	case User:
		_, ok := s.users[target]
		return ok, nil
	default:
		return false, xerrors.ErrInvalidSessionKind
	}
}

// Bind 绑定用户ID
// Bind 绑定用户ID，如果 uid 已经绑定旧连接，则挤掉旧连接
func (s *Session) Bind(cid, uid int64) error {
	var kickConn network.Conn

	s.rw.Lock()

	conn, err := s.connLocked(Conn, cid)
	if err != nil {
		s.rw.Unlock()
		return err
	}

	oldUID := conn.UID()
	if oldUID == uid {
		s.rw.Unlock()
		return nil
	}

	// 删除当前连接旧 uid 映射
	if oldUID != 0 {
		if curConn, ok := s.users[oldUID]; ok && curConn.ID() == cid {
			delete(s.users, oldUID)
		}
	}

	// uid = 0，当作解绑
	if uid == 0 {
		conn.Unbind()
		s.rw.Unlock()
		return nil
	}

	// 这个 uid 已经被其他连接绑定，准备挤掉旧连接
	if oldConn, ok := s.users[uid]; ok {
		if oldConn.ID() == cid {
			conn.Bind(uid)
			s.users[uid] = conn
			s.rw.Unlock()
			return nil
		}

		// 先解绑旧连接，防止旧连接关闭时 RemConn 误删 users[uid]
		oldConn.Unbind()

		// 不要在锁内 Close，先记录下来
		kickConn = oldConn
	}

	// 绑定新连接
	conn.Bind(uid)
	s.users[uid] = conn

	s.rw.Unlock()

	// 锁外关闭旧连接，避免死锁
	if kickConn != nil {
		go func(c network.Conn) {
			// Graceful close flushes the replacement notification already queued
			// on the high-priority write channel before sending the close signal.
			_ = c.Close()
		}(kickConn)
	}

	return nil
}

// Unbind 解绑用户ID
//
// 安全原则：
// 1. 必须通过 cid 找到指定连接
// 2. 必须校验 conn.UID() == uid
// 3. uid 不匹配说明用户可能已经重连，不能解绑
// 4. 只删除 users[uid] 当前仍然指向 cid 的映射，避免误删新连接
func (s *Session) Unbind(cid, uid int64) (int64, error) {
	if uid == 0 || cid == 0 {
		return 0, xerrors.ErrNotFoundSession
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	conn, err := s.connLocked(Conn, cid)
	if err != nil {
		return 0, err
	}

	// cid 对应的连接已经不是这个 uid，说明可能已经重连/解绑
	// 不能继续操作，避免误伤其他连接
	if conn.UID() != uid {
		return 0, nil
	}

	conn.Unbind()

	// 只有 users[uid] 当前还指向这个 cid，才删除
	if curConn, ok := s.users[uid]; ok && curConn != nil && curConn.ID() == cid {
		delete(s.users, uid)
	}

	return cid, nil
}

// LocalIP 获取本地IP
func (s *Session) LocalIP(kind Kind, target int64) (string, error) {
	conn, err := s.getConn(kind, target)
	if err != nil {
		return "", err
	}

	return conn.LocalIP()
}

// LocalAddr 获取本地地址
func (s *Session) LocalAddr(kind Kind, target int64) (net.Addr, error) {
	conn, err := s.getConn(kind, target)
	if err != nil {
		return nil, err
	}

	return conn.LocalAddr()
}

// RemoteIP 获取远端IP
func (s *Session) RemoteIP(kind Kind, target int64) (string, error) {
	conn, err := s.getConn(kind, target)
	if err != nil {
		return "", err
	}

	return conn.RemoteIP()
}

// RemoteAddr 获取远端地址
func (s *Session) RemoteAddr(kind Kind, target int64) (net.Addr, error) {
	conn, err := s.getConn(kind, target)
	if err != nil {
		return nil, err
	}

	return conn.RemoteAddr()
}

// 安全原则：
// 1. 必须先通过 cid 找连接
// 2. 必须校验 conn.UID() == uid
// 3. uid 不匹配说明连接可能已经重连 / 被复用 / 已解绑，不能关闭
// 4. 不要在 Session 锁内调用 conn.Close，避免 Close -> RemConn -> 再次拿锁导致死锁
func (s *Session) Close(cid, uid int64, force ...bool) error {
	var conn network.Conn

	s.rw.RLock()

	c, ok := s.conns[cid]
	if !ok || c == nil {
		s.rw.RUnlock()
		return xerrors.ErrNotFoundSession
	}

	// 必须校验 cid 对应的连接是否还是这个 uid
	// 不匹配说明用户可能已经重连，不能按 uid 去找新连接关闭，否则会误伤。
	if c.UID() != uid {
		s.rw.RUnlock()
		return nil
	}

	conn = c

	s.rw.RUnlock()

	// 锁外关闭连接
	// 关闭后触发 RemConn，由 RemConn 统一清理 conns/users/channels
	return conn.Close(force...)
}

// Send 发送消息，同步
func (s *Session) Send(kind Kind, target int64, message []byte) error {
	conn, err := s.getConn(kind, target)
	if err != nil {
		return err
	}

	// 不要在 Session 锁内做网络写
	return conn.Send(message)
}

// Push 推送消息，异步
func (s *Session) Push(kind Kind, target int64, message []byte) error {
	conn, err := s.getConn(kind, target)
	if err != nil {
		return err
	}

	// 不要在 Session 锁内做网络写
	return conn.Push(message)
}

// Multicast 推送组播消息，异步
func (s *Session) Multicast(kind Kind, targets []int64, message []byte) (int64, error) {
	if len(targets) == 0 {
		return 0, nil
	}

	conns, err := s.snapshotTargets(kind, targets)
	if err != nil {
		return 0, err
	}

	var n int64
	for _, conn := range conns {
		if conn.Push(message) == nil {
			n++
		}
	}

	return n, nil
}

// Broadcast 推送广播消息，异步
func (s *Session) Broadcast(kind Kind, message []byte) (int64, error) {
	conns, err := s.snapshotAll(kind)
	if err != nil {
		return 0, err
	}

	var n int64
	for _, conn := range conns {
		if conn.Push(message) == nil {
			n++
		}
	}

	return n, nil
}

// Publish 发布频道消息，异步
func (s *Session) Publish(channel string, message []byte) int64 {
	if channel == "" {
		return 0
	}

	conns := s.snapshotChannel(channel)

	var n int64
	for _, conn := range conns {
		if conn.Push(message) == nil {
			n++
		}
	}

	return n
}

// Subscribe 订阅频道
func (s *Session) Subscribe(kind Kind, targets []int64, channel string) error {
	if len(targets) == 0 || channel == "" {
		return nil
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	conns, err := s.sessionsLocked(kind)
	if err != nil {
		return err
	}

	channelConns, ok := s.channels[channel]
	if !ok {
		channelConns = make(map[network.Conn]struct{}, len(targets))
		s.channels[channel] = channelConns
	}

	attrKey := channelAttrKey(channel)

	for _, target := range targets {
		conn, ok := conns[target]
		if !ok || conn == nil {
			continue
		}

		conn.Attr().Set(attrKey, struct{}{})
		channelConns[conn] = struct{}{}
	}

	return nil
}

// Unsubscribe 取消订阅频道
func (s *Session) Unsubscribe(kind Kind, targets []int64, channel string) error {
	if len(targets) == 0 || channel == "" {
		return nil
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	conns, err := s.sessionsLocked(kind)
	if err != nil {
		return err
	}

	attrKey := channelAttrKey(channel)

	for _, target := range targets {
		conn, ok := conns[target]
		if !ok || conn == nil {
			continue
		}

		if ok = conn.Attr().Del(attrKey); ok {
			s.doUnsubscribeLocked(channel, conn)
		}
	}

	return nil
}

// Stat 统计会话总数
func (s *Session) Stat(kind Kind) (int64, error) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	switch kind {
	case Conn:
		return int64(len(s.conns)), nil
	case User:
		return int64(len(s.users)), nil
	default:
		return 0, xerrors.ErrInvalidSessionKind
	}
}

// getConn 获取连接，拿到连接后立即释放 Session 锁
func (s *Session) getConn(kind Kind, target int64) (network.Conn, error) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	return s.connLocked(kind, target)
}

// snapshotTargets 快照目标连接。
// 注意：这里只在锁内取连接快照，不在锁内 Push。
func (s *Session) snapshotTargets(kind Kind, targets []int64) ([]network.Conn, error) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	conns, err := s.sessionsLocked(kind)
	if err != nil {
		return nil, err
	}

	result := make([]network.Conn, 0, len(targets))
	for _, target := range targets {
		conn, ok := conns[target]
		if !ok || conn == nil {
			continue
		}

		result = append(result, conn)
	}

	return result, nil
}

// snapshotAll 快照全部连接。
// 注意：这里只在锁内取连接快照，不在锁内 Push。
func (s *Session) snapshotAll(kind Kind) ([]network.Conn, error) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	conns, err := s.sessionsLocked(kind)
	if err != nil {
		return nil, err
	}

	result := make([]network.Conn, 0, len(conns))
	for _, conn := range conns {
		if conn != nil {
			result = append(result, conn)
		}
	}

	return result, nil
}

// snapshotChannel 快照频道连接。
// 注意：这里只在锁内取连接快照，不在锁内 Push。
func (s *Session) snapshotChannel(channel string) []network.Conn {
	s.rw.RLock()
	defer s.rw.RUnlock()

	channelConns, ok := s.channels[channel]
	if !ok || len(channelConns) == 0 {
		return nil
	}

	result := make([]network.Conn, 0, len(channelConns))
	for conn := range channelConns {
		if conn != nil {
			result = append(result, conn)
		}
	}

	return result
}

// sessionsLocked 获取会话 map。
// 调用方必须已经持有锁。
func (s *Session) sessionsLocked(kind Kind) (map[int64]network.Conn, error) {
	switch kind {
	case Conn:
		return s.conns, nil
	case User:
		return s.users, nil
	default:
		return nil, xerrors.ErrInvalidSessionKind
	}
}

// connLocked 获取连接。
// 调用方必须已经持有锁。
func (s *Session) connLocked(kind Kind, target int64) (network.Conn, error) {
	conns, err := s.sessionsLocked(kind)
	if err != nil {
		return nil, err
	}

	conn, ok := conns[target]
	if !ok || conn == nil {
		return nil, xerrors.ErrNotFoundSession
	}

	return conn, nil
}

// doUnsubscribeLocked 取消订阅频道。
// 调用方必须已经持有写锁。
func (s *Session) doUnsubscribeLocked(channel string, conn network.Conn) {
	channelConns, ok := s.channels[channel]
	if !ok {
		return
	}

	delete(channelConns, conn)

	if len(channelConns) == 0 {
		delete(s.channels, channel)
	}
}

func channelAttrKey(channel string) string {
	return channelAttrPrefix + channel
}
