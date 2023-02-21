package common

import (
	"go.uber.org/zap"
	"net"
)

type IdleTimeoutConnV3 struct {
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

	go ic.UpdateIdleTime()
	select {
	case ic.Updated <- 1:
	default:
	}
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
	go ic.UpdateIdleTime()
	select {
	case ic.Updated <- 1:
	default:
	}
	return ic.Conn.Write(buf)
}

func (c *IdleTimeoutConnV3) Close() {
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
}
