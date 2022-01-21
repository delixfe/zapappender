package zapappender

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/delixfe/zapappender/internal/bufferpool"
	"go.uber.org/multierr"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// TODO: message structs could be used in general
type writeMessage struct {
	// TODO: create a custom []byte buffer instance so we do not need to keep the reference to the pool?
	buf   *buffer.Buffer
	ent   zapcore.Entry
	flush chan struct{}
}

var ErrAppenderShutdown = errors.New("appender shut down")

var _ SynchronizationAwareAppender = &Async{}

type Async struct {
	// only during construction
	maxQueueLength           int
	calculateDropThresholdFn func(*Async) (int, error)

	// readonly
	primary           Appender
	fallback          Appender
	monitorPeriod     time.Duration
	fallbackThreshold int
	syncTimeout       time.Duration

	// state
	queueWrite chan writeMessage
	close      chan struct{}
	shutdown   int32 // incremented by Shutdown
}

func NewAsync(primary Appender, options ...AsyncOption) (a *Async, err error) {
	if primary == nil {
		return nil, errors.New("primary is required")
	}
	a = &Async{
		primary: primary,
	}

	AsyncMaxQueueLength(1000).apply(a)
	AsyncQueueMonitorPeriod(time.Second).apply(a)
	AsyncQueueMinFreePercent(.1).apply(a)
	AsyncOnQueueNearlyFullDropMessages().apply(a)

	for _, option := range options {
		err = option.apply(a)
		if err != nil {
			return nil, err
		}
	}

	a.queueWrite = make(chan writeMessage, a.maxQueueLength)
	a.fallbackThreshold, err = a.calculateDropThresholdFn(a)
	a.close = make(chan struct{})

	a.start()

	return a, err
}

func (a *Async) start() {
	go a.forwardWrite()
	go a.monitorQueueWrite()
}

// the return value n does not work in an async context
func (a *Async) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	if atomic.LoadInt32(&a.shutdown) != 0 {
		err = ErrAppenderShutdown
		return
	}

	msg := writeMessage{
		buf: bufferpool.Get(),
		ent: ent,
	}

	n, err = msg.buf.Write(p)
	if err != nil {
		return
	}

	// this might block shortly until the monitoring routine drops messages
	a.queueWrite <- msg
	return
}

func (m *writeMessage) flushMarker() bool {
	if m.flush == nil {
		return false
	}
	close(m.flush)
	return true
}

func (a *Async) forwardWrite() {
	for {
		select {
		case <-a.close:
			return
		case msg := <-a.queueWrite:
			if msg.flushMarker() {
				continue
			}
			// TODO: handle error
			_, _ = a.primary.Write(msg.buf.Bytes(), msg.ent)
			msg.buf.Free()
		}
	}
}

func (a *Async) monitorQueueWrite() {
	ticker := time.NewTicker(a.monitorPeriod)
	for {
		select {
		case <-ticker.C:
		case <-a.close:
			return
		}
		available := cap(a.queueWrite) - len(a.queueWrite)
		toFree := a.fallbackThreshold - available
		for i := 0; i < toFree; i++ {
			select {
			case <-a.close:
				return
			case msg := <-a.queueWrite:
				if msg.flushMarker() {
					continue
				}
				// TODO: drop or Fallback: add messageFullStrategy
				a.fallback.Write(msg.buf.Bytes(), msg.ent)
				msg.buf.Free()
			}
		}
	}
}

func (a *Async) Sync() error {
	ctx := context.Background()
	if a.syncTimeout != time.Duration(0) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.syncTimeout)
		defer cancel()
	}
	a.Drain(ctx)
	return multierr.Append(a.primary.Sync(), a.fallback.Sync())
}

// Drain tries to gracefully drain the remaining buffered messages,
// blocking until the buffer is empty or the provided context is cancelled.
func (a *Async) Drain(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-ctx.Done():
		return
	default:
	}
	// TODO: also we could use Fallback to drain. add to messageFullStrategy interface
	done := make(chan struct{})
	msg := writeMessage{
		flush: done,
	}
	a.queueWrite <- msg
	select {
	case <-ctx.Done(): // we timed out
	case <-done: // our marker message was handled
	}
}

func (a *Async) Synchronized() bool {
	return true
}

func (a *Async) Shutdown(ctx context.Context) {
	if atomic.SwapInt32(&a.shutdown, 1) != 0 {
		return // already called
	}

	a.Drain(ctx)
	close(a.close) // stop the loops, after draining
	close(a.queueWrite)
}
