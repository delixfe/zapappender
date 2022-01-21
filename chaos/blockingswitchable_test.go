package chaos

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/delixfe/zapappender"

	"go.uber.org/zap/zapcore"
)

func TestBlockingSwitchable_Break(t *testing.T) {
	written := uint32(0)
	ctx := context.Background()

	inner := zapappender.NewDelegating(func(p []byte, ent zapcore.Entry) (n int, err error) {
		atomic.AddUint32(&written, 1)
		return len(p), nil
	}, nil, true)

	blocking := NewBlockingSwitchableCtx(ctx, inner)
	blocking.Write([]byte{}, zapcore.Entry{})
	if written != 1 {
		t.Fatal("expected 1 write")
	}

	blocking.Break()

	go func() {
		blocking.Write([]byte{}, zapcore.Entry{})
	}()

	time.Sleep(time.Millisecond * 100)
	if atomic.LoadUint32(&written) != 1 {
		t.Fatal("expected no further write while blocking")
	}

	blocking.Fix()

	time.Sleep(time.Millisecond * 100)

	if atomic.LoadUint32(&written) != 2 {
		t.Errorf("expected 2nd write after unblocking, written is %d", written)
	}
}
