package signal

import (
	"context"
	"github.com/gotcpforward/gotcpforward/task"
	"go.uber.org/zap"
	"sync"
	"time"
)

type ActivityUpdater interface {
	Update()
}

type ActivityTimer struct {
	sync.RWMutex
	updated   chan struct{}
	checkTask *task.Periodic
	onTimeout func()
	logger    *zap.SugaredLogger
}

func (t *ActivityTimer) Update() {
	cm := "Update@timer.go"
	select {
	case t.updated <- struct{}{}:
		t.logger.Infof(cm + " update timer for ActivityTimer")
	default:
	}
}

func (t *ActivityTimer) check() error {
	select {
	case <-t.updated:
	default:
		t.finish()
	}
	return nil
}

func (t *ActivityTimer) finish() {
	t.Lock()
	defer t.Unlock()

	if t.onTimeout != nil {
		t.onTimeout()
		t.onTimeout = nil
	}
	if t.checkTask != nil {
		t.checkTask.Close()
		t.checkTask = nil
	}
}

func (t *ActivityTimer) SetTimeout(timeout time.Duration) {
	if timeout == 0 {
		t.finish()
		return
	}

	checkTask := &task.Periodic{
		Interval: timeout,
		Execute:  t.check,
	}

	t.Lock()

	if t.checkTask != nil {
		t.checkTask.Close()
	}
	t.checkTask = checkTask
	t.Unlock()
	t.Update()
	err := checkTask.Start()
	if err != nil {
		panic(err)
	}
}

func CancelAfterInactivity(ctx context.Context, cancel func(), timeout time.Duration, logger *zap.SugaredLogger) *ActivityTimer {
	timer := &ActivityTimer{
		updated:   make(chan struct{}, 1),
		onTimeout: cancel,
		logger:    logger,
	}
	timer.SetTimeout(timeout)
	return timer
}
