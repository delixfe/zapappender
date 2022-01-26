package chaos

import (
	"errors"

	"github.com/delixfe/zapappender"
	"go.uber.org/zap/zapcore"
)

var ErrFailEnabled = errors.New("failing switchable appender is failing")

var (
	_ zapappender.Appender = &FailingSwitchable{}
	_ Switchable           = &FailingSwitchable{}
)

// FailingSwitchable returns an error on all writes while it is Breaking.
type FailingSwitchable struct {
	primary zapappender.Appender
	enabled bool
}

func NewFailingSwitchable(inner zapappender.Appender) *FailingSwitchable {
	return &FailingSwitchable{
		primary: inner,
		enabled: false,
	}
}

// Breaking returns true if FailingSwitchable is set to fail.
func (a *FailingSwitchable) Breaking() bool {
	return a.enabled
}

// Break starts failing messages.
func (a *FailingSwitchable) Break() {
	a.enabled = true
}

// Fix stops failing messages.
func (a *FailingSwitchable) Fix() {
	a.enabled = false
}

func (a *FailingSwitchable) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	if a.enabled {
		return 0, ErrFailEnabled
	}
	n, err = a.primary.Write(p, ent)
	if err == nil {
		return
	}
	return

}

func (a *FailingSwitchable) Sync() error {
	return a.primary.Sync()
}
