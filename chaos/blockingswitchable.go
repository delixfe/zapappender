package chaos

import (
	"context"

	"github.com/delixfe/zapappender"
	"go.uber.org/zap/zapcore"
)

var (
	_ zapappender.Appender = &BlockingSwitchable{}
	_ Switchable           = &BlockingSwitchable{}
)

type BlockingSwitchable struct {
	primary zapappender.Appender
	enabled bool
	waiting chan struct{}
	ctx     context.Context
}

func NewBlockingSwitchable(inner zapappender.Appender) *BlockingSwitchable {
	return NewBlockingSwitchableCtx(nil, inner)
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

func (a *BlockingSwitchable) Breaking() bool {
	return a.enabled
}

func (a *BlockingSwitchable) Break() {
	if a.enabled {
		return
	}
	a.enabled = true
	a.waiting = make(chan struct{})
}

func (a *BlockingSwitchable) Fix() {
	a.enabled = false
	close(a.waiting)
}

func (a *BlockingSwitchable) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	if a.enabled {
		select {
		case <-a.ctx.Done():
			return 0, a.ctx.Err()
		case <-a.waiting:
		}
	}
	n, err = a.primary.Write(p, ent)
	return

}

func (a *BlockingSwitchable) Sync() error {
	return a.primary.Sync()
}
