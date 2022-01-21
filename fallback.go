package zapappender

import (
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
)

var _ SynchronizationAwareAppender = &Fallback{}

type Fallback struct {
	primary   Appender
	secondary Appender
}

// NewFallback forwards the message to secondary, if writing to primary returned an error.
// secondary is wrapped in a Synchronizing zapappender.
func NewFallback(primary, secondary Appender) *Fallback {
	return &Fallback{
		primary:   primary,
		secondary: NewSynchronizing(secondary),
	}
}

func (a *Fallback) Write(p []byte, ent zapcore.Entry) (n int, err error) {

	n, primErr := a.primary.Write(p, ent)
	if primErr == nil {
		return n, nil
	}
	n, fallErr := a.secondary.Write(p, ent)
	if fallErr == nil {
		return n, nil
	}

	// TODO: decide which error to return
	return n, multierr.Append(primErr, fallErr)

}

func (a *Fallback) Sync() error {
	return multierr.Append(a.primary.Sync(), a.secondary.Sync())
}

func (a *Fallback) Synchronized() bool {
	return Synchronized(a.primary)
}
