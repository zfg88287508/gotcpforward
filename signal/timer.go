package signal

import (
	"context"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
	"time"
)

type ActivityUpdater interface {
	Update()
}

type ActivityTimer struct {
	sync.RWMutex
	updated     atomic.Int64
	onTimeout   func()
	timerClosed bool
	updateLock  sync.Mutex
	tTimeout    time.Duration
	logger      *zap.SugaredLogger
}

func (t *ActivityTimer) Update() {
	tsn := time.Now().Add(t.tTimeout).Unix()
	t.logger.Infof("update timer for ActivityTimer:%v \n", tsn)
	go t.updated.Swap(tsn)
}

func (t *ActivityTimer) check() {
	ttn := t.updated.Load()
	if ttn <= 0 || ttn < time.Now().Unix() {
		t.finish()
	}
}

func (t *ActivityTimer) finish() {
	t.Lock()
	defer t.Unlock()

	t.timerClosed = true
	if t.onTimeout != nil {
		t.onTimeout()
		t.onTimeout = nil
	}
}

func (t *ActivityTimer) SetTimeout(timeout time.Duration) {
	if timeout == 0 {
		t.finish()
		return
	}

	//过N 秒，执行一次 check
	t.Update()
	go func() {
		for {
			if t.timerClosed {
				t.logger.Infof("ActivityTimer finish and close\n")
				break
			}
			time.Sleep(timeout)
			t.check()
		}
	}()
}

func CancelAfterInactivity(ctx context.Context, cancel func(), timeout time.Duration, logger *zap.SugaredLogger) *ActivityTimer {
	timer := &ActivityTimer{
		updated:   atomic.Int64{},
		onTimeout: cancel,
		logger:    logger,
	}
	timer.SetTimeout(timeout)
	return timer
}
