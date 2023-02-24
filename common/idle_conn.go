package common

import (
	"errors"
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
	case <-time.After(100 * time.Millisecond):

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
	case <-time.After(100 * time.Millisecond):
	}

	if ic.Conn != nil {
		return ic.Conn.Read(buf)
	}
	return 0, errors.New(" IdleTimeoutConnV3 failed to Read")
}

func (ic *IdleTimeoutConnV3) UpdateIdleTime() {
	select {
	case <-ic.Updated:

		la := ic.AfterDelay.Load()
		ic.AfterDelay.Store(time.Now().Add(600 * time.Millisecond).UnixMilli())
		if la <= 0 || la <= time.Now().UnixMilli() {
			go ic.update()
		}
	case <-time.After(200 * time.Millisecond):

	}
}

func (ic *IdleTimeoutConnV3) Write(buf []byte) (int, error) {
	go ic.UpdateIdleTime()
	select {
	case ic.Updated <- true:
	case <-time.After(100 * time.Millisecond):
	}
	if ic.Conn != nil {
		return ic.Conn.Write(buf)
	}
	return 0, errors.New(" IdleTimeoutConnV3 failed to Write")
}

func (c *IdleTimeoutConnV3) Close() {
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
}
