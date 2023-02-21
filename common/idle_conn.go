package common

import (
	"go.uber.org/zap"
	"net"
	"sync/atomic"
	"time"
)

type IdleTimeoutConnV3 struct {
	update     func()
	Conn       net.Conn
	logger     *zap.SugaredLogger
	Updated    chan bool
	AfterDelay atomic.Int64
}

func NewIdleTimeoutConnV3(conn net.Conn, fn func(), logger *zap.SugaredLogger) *IdleTimeoutConnV3 {
	ch := make(chan bool)
	select {
	case ch <- true:
	default:

	}
	c := &IdleTimeoutConnV3{
		Conn:    conn,
		update:  fn,
		logger:  logger,
		Updated: ch,
	}
	return c
}

func (ic *IdleTimeoutConnV3) Read(buf []byte) (int, error) {

	go ic.UpdateIdleTime()
	select {
	case ic.Updated <- true:
	default:
	}
	return ic.Conn.Read(buf)
}

func (ic *IdleTimeoutConnV3) UpdateIdleTime() {
	select {
	case <-ic.Updated:

		la := ic.AfterDelay.Load()
		ic.AfterDelay.Store(time.Now().Add(5 * time.Second).Unix())
		if la <= 0 || la <= time.Now().Unix() {
			go ic.update()
		}
	case <-time.After(1 * time.Second):

	}
}

func (ic *IdleTimeoutConnV3) Write(buf []byte) (int, error) {
	go ic.UpdateIdleTime()
	select {
	case ic.Updated <- true:
	default:
	}
	return ic.Conn.Write(buf)
}

func (c *IdleTimeoutConnV3) Close() {
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
}
