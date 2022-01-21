package zapappender

import (
	"io"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

type AsyncOptions []AsyncOption
type assertFn func(*Async) bool

func TestNewAsync(t *testing.T) {

	tests := []struct {
		name       string
		options    AsyncOptions
		wantErr    bool
		assertions []assertFn
	}{
		{name: "forwardTo nil fallback", wantErr: true,
			options: AsyncOptions{AsyncOnQueueNearlyFullForwardTo(nil)}},
		{name: "forwardTo fallback is synchronized",
			options:    AsyncOptions{AsyncOnQueueNearlyFullForwardTo(NewWriter(zapcore.AddSync(io.Discard)))},
			assertions: []assertFn{func(a *Async) bool { return Synchronized(a) }},
		},
		{name: "max queue length negative", wantErr: true, options: AsyncOptions{AsyncMaxQueueLength(-1)}},
		{name: "queue monitor period negative", wantErr: true, options: AsyncOptions{AsyncQueueMonitorPeriod(-1 * time.Second)}},
		{name: "queue monitor period zero", wantErr: true, options: AsyncOptions{AsyncQueueMonitorPeriod(0)}},
		{name: "async sync timeout negative", wantErr: true, options: AsyncOptions{AsyncSyncTimeout(-1 * time.Second)}},
		{name: "sync sync timeout zero", wantErr: true, options: AsyncOptions{AsyncSyncTimeout(0)}},
		{name: "min free items lt zero", wantErr: true, options: AsyncOptions{
			AsyncQueueMinFreeItems(-1),
		}},
		{name: "min free items greater queue length", wantErr: true, options: AsyncOptions{
			AsyncQueueMinFreeItems(100),
			AsyncMaxQueueLength(10),
		}},
		{name: "min free percent lt 0", wantErr: true, options: AsyncOptions{
			AsyncQueueMinFreePercent(-1)}},
		{name: "min free percent gt 1", wantErr: true, options: AsyncOptions{
			AsyncQueueMinFreePercent(-1)}},

		{name: "min free percent calculation 10",
			options: AsyncOptions{
				AsyncQueueMinFreePercent(0.1),
				AsyncMaxQueueLength(100),
			},
			assertions: []assertFn{func(a *Async) bool { return a.fallbackThreshold == 10 }},
		},
		{name: "min free percent calculation 20",
			options: AsyncOptions{
				AsyncQueueMinFreePercent(0.2),
				AsyncMaxQueueLength(10),
			},
			assertions: []assertFn{func(a *Async) bool { return a.fallbackThreshold == 2 }},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary := NewDiscard()
			gotA, err := NewAsync(primary, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAsync() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, assertion := range tt.assertions {
				if !assertion(gotA) {
					t.Error("assertion failed")
				}
			}
		})
	}
}

func TestNewAsync_nilPrimary_returnsErr(t *testing.T) {
	async, err := NewAsync(nil)
	if err == nil {
		t.Error("expected an error")
	}
	if async != nil {
		t.Errorf("did not expcect an appender")
	}
}
