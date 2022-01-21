package zapappender

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = &AppenderCore{}

// Appender is the interface for composable appenders.
//
// The Write method receives the zapcore.Entry in addition to the text buffer.
// This allows appenders access to the fields like Time.
//
// Several variants of the interface were analyzed.
// 1. Write with p, ent, fields
// 2. Write with p, ent
// 3. Write with p and a subset of ent
// A. Append with enc, ent, fields
//
// Decision: variant 2 - thus variant 3 would also be an option.
// - we cannot keep the fields in an async process
//	- they might hold references that might be already mutated or hinder GC
// - without fields, we cannot use the Encoder to encode the message
type Appender interface {

	// Write
	// must not retain p
	Write(p []byte, ent zapcore.Entry) (n int, err error)

	// Sync flushes buffered logs (if any).
	Sync() error
}

type SynchronizationAware interface {
	Synchronized() bool
}

type SynchronizationAwareAppender interface {
	Appender
	SynchronizationAware
}

func Synchronized(s interface{}) bool {
	if s, ok := s.(SynchronizationAware); ok && s.Synchronized() {
		return true
	}
	return false
}

var _ SynchronizationAwareAppender = &Synchronizing{}

type Synchronizing struct {
	primary Appender
	mutex   sync.Mutex
}

func NewSynchronizing(inner Appender) Appender {
	if inner == nil {
		return nil
	}
	if Synchronized(inner) {
		// already synchronizing
		return inner
	}
	return &Synchronizing{
		primary: inner,
	}
}

func (s *Synchronizing) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.primary.Write(p, ent)
}

func (s *Synchronizing) Sync() error {
	//TODO: should we lock Sync?
	return s.primary.Sync()
}

func (s *Synchronizing) Synchronized() bool {
	return true
}

var _ zapcore.Core = &AppenderCore{}

// AppenderCore bridges between zapcore and zapappender.
type AppenderCore struct {
	zapcore.LevelEnabler
	enc      zapcore.Encoder
	appender Appender
}

func NewAppenderCore(enc zapcore.Encoder, appender Appender, enab zapcore.LevelEnabler) *AppenderCore {

	return &AppenderCore{
		LevelEnabler: enab,
		enc:          enc,
		appender:     NewSynchronizing(appender),
	}
}

func (c *AppenderCore) With(fields []zapcore.Field) zapcore.Core {
	enc := c.enc.Clone()
	for i := range fields {
		fields[i].AddTo(enc)
	}
	return &AppenderCore{
		LevelEnabler: c.LevelEnabler,
		appender:     c.appender,
		enc:          enc,
	}
}

func (c *AppenderCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *AppenderCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}
	_, err = c.appender.Write(buf.Bytes(), ent)
	buf.Free()
	if err != nil {
		return err
	}
	if ent.Level > zapcore.ErrorLevel {
		// Since we may be crashing the program, sync the output. Ignore Sync
		// errors, pending a clean solution to issue #370.
		_ = c.Sync()
	}
	return nil
}

func (c *AppenderCore) Sync() error {
	return c.appender.Sync()
}
