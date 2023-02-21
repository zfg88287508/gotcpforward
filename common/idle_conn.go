package common

import (
	"go.uber.org/zap"
	"net"
	"sync"
	"sync/atomic"
)

type IdleTimeoutConnV3 struct {
	sync.Mutex
	atomic.Int64
	update  func()
	Conn    net.Conn
	logger  *zap.SugaredLogger
	Updated chan int
}

func NewIdleTimeoutConnV3(conn net.Conn, fn func(), logger *zap.SugaredLogger) *IdleTimeoutConnV3 {
	c := &IdleTimeoutConnV3{
		Conn:    conn,
		update:  fn,
		logger:  logger,
		Updated: make(chan int),
	}
	return c
}

func (ic *IdleTimeoutConnV3) Read(buf []byte) (int, error) {
	select {
	case ic.Updated <- 1:
	default:
	}
	go ic.UpdateIdleTime()
	return ic.Conn.Read(buf)
}

func (ic *IdleTimeoutConnV3) UpdateIdleTime() {
	select {
	case <-ic.Updated:
		go ic.update()
	default:
	}
}

func (ic *IdleTimeoutConnV3) Write(buf []byte) (int, error) {
	select {
	case ic.Updated <- 1:
	default:
	}
	go ic.UpdateIdleTime()
	return ic.Conn.Write(buf)
}

func (c *IdleTimeoutConnV3) Close() {
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
}
