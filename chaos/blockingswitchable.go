package chaos

import (
	"context"
	"sync"

	"github.com/delixfe/zapappender"
	"go.uber.org/zap/zapcore"
)

var (
	_ zapappender.Appender = &BlockingSwitchable{}
	_ Switchable           = &BlockingSwitchable{}
)

// BlockingSwitchable allows to block all messages until it is released.
type BlockingSwitchable struct {
	primary zapappender.Appender
	enabled bool
	waiting chan struct{}
	ctx     context.Context
	mu      sync.Mutex
}

func NewBlockingSwitchable(inner zapappender.Appender) *BlockingSwitchable {
	return NewBlockingSwitchableCtx(context.Background(), inner)
}

func NewBlockingSwitchableCtx(ctx context.Context, inner zapappender.Appender) *BlockingSwitchable {
	if ctx == nil {
		ctx = context.Background()
	}
	return &BlockingSwitchable{
		primary: inner,
		enabled: false,
		ctx:     ctx,
	}
}

// Breaking returns true if messages are currently blocked.
func (a *BlockingSwitchable) Breaking() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.enabled
}

// Break blocks all messages until Fix is called.
func (a *BlockingSwitchable) Break() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.enabled {
		return
	}
	a.enabled = true
	a.waiting = make(chan struct{})
}

// Fix unblocks the messages.
func (a *BlockingSwitchable) Fix() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = false
	close(a.waiting)
}

func (a *BlockingSwitchable) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	a.mu.Lock()
	enabled := a.enabled
	waiting := a.waiting
	a.mu.Unlock()
	if enabled {
		select {
		case <-a.ctx.Done():
			return 0, a.ctx.Err()
		case <-waiting:
		}
	}
	n, err = a.primary.Write(p, ent)
	return

}

func (a *BlockingSwitchable) Sync() error {
	return a.primary.Sync()
}
