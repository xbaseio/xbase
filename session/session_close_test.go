package session

import (
	"net"
	"testing"

	"github.com/xbaseio/xbase/network"
)

type closeTestConn struct {
	id     int64
	uid    int64
	closed bool
	force  bool
}

func (c *closeTestConn) ID() int64                { return c.id }
func (c *closeTestConn) UID() int64               { return c.uid }
func (c *closeTestConn) Attr() network.Attr       { return closeTestAttr{} }
func (c *closeTestConn) Bind(uid int64)           { c.uid = uid }
func (c *closeTestConn) Unbind()                  { c.uid = 0 }
func (c *closeTestConn) Send([]byte) error        { return nil }
func (c *closeTestConn) Push([]byte) error        { return nil }
func (c *closeTestConn) State() network.ConnState { return network.ConnOpened }
func (c *closeTestConn) Close(force ...bool) error {
	c.closed = true
	c.force = len(force) > 0 && force[0]
	return nil
}
func (c *closeTestConn) LocalIP() (string, error)      { return "", nil }
func (c *closeTestConn) LocalAddr() (net.Addr, error)  { return nil, nil }
func (c *closeTestConn) RemoteIP() (string, error)     { return "", nil }
func (c *closeTestConn) RemoteAddr() (net.Addr, error) { return nil, nil }

type closeTestAttr struct{}

func (closeTestAttr) Set(any, any)              {}
func (closeTestAttr) Get(any) (any, bool)       { return nil, false }
func (closeTestAttr) Del(any) bool              { return false }
func (closeTestAttr) Visit(func(any, any) bool) {}

func TestSessionCloseFallsBackToUIDWhenCIDMissing(t *testing.T) {
	conn := &closeTestConn{id: 10, uid: 10003}
	s := NewSession()
	s.conns[conn.id] = conn
	s.users[conn.uid] = conn

	if err := s.Close(9, conn.uid, true); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !conn.closed || !conn.force {
		t.Fatalf("current UID connection was not force closed: %+v", conn)
	}
}

func TestSessionCloseDoesNotFallbackWhenCIDBelongsToAnotherUID(t *testing.T) {
	cidConn := &closeTestConn{id: 9, uid: 20000}
	uidConn := &closeTestConn{id: 10, uid: 10003}
	s := NewSession()
	s.conns[cidConn.id] = cidConn
	s.conns[uidConn.id] = uidConn
	s.users[uidConn.uid] = uidConn

	if err := s.Close(cidConn.id, uidConn.uid, true); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if cidConn.closed || uidConn.closed {
		t.Fatalf("Close affected a connection after CID/UID mismatch: cid=%+v uid=%+v", cidConn, uidConn)
	}
}
