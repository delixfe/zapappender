package zapappender

import (
	"go.uber.org/zap/zapcore"
)

var _ SynchronizationAwareAppender = &Discard{}

// Discard silently drops all messages
type Discard struct {
}

func NewDiscard() *Discard {
	return &Discard{}
}

func (a *Discard) Write(p []byte, _ zapcore.Entry) (int, error) {
	return len(p), nil
}

func (a *Discard) Sync() error {
	return nil
}

func (a *Discard) Synchronized() bool {
	return true
}
