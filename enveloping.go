package zapappender

import (
	"github.com/delixfe/zapappender/internal/bufferpool"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// EnvelopingFn Function to create the enveloped output.
// p contains the original content
// the enveloped content must be written into output
// entry by ref or pointer?
// -> benchmarks show that using a pointer creates one alloc (for the pointer)
//    but passing by value does not
type EnvelopingFn func(p []byte, ent zapcore.Entry, output *buffer.Buffer) error

var _ SynchronizationAwareAppender = &Enveloping{}

// Enveloping allows to adapt the log message.
// This can be used to format the message output. That is especially usefull when a format should only
// be applied to a primary appender but not a fallback one.
type Enveloping struct {
	primary Appender
	envFn   EnvelopingFn
}

func (a *Enveloping) Synchronized() bool {
	return Synchronized(a.primary)
}

func NewEnveloping(inner Appender, envFn EnvelopingFn) *Enveloping {
	return &Enveloping{
		primary: inner,
		envFn:   envFn,
	}
}

func NewEnvelopingPreSuffix(inner Appender, prefix, suffix string) *Enveloping {
	envFn := func(p []byte, ent zapcore.Entry, output *buffer.Buffer) error {
		output.WriteString(prefix)
		output.Write(p)
		output.WriteString(suffix)
		return nil
	}
	return NewEnveloping(inner, envFn)
}

func (a *Enveloping) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	buf := bufferpool.Get()
	defer buf.Free()
	err = a.envFn(p, ent, buf)
	if err != nil {
		return
	}
	n, err = a.primary.Write(buf.Bytes(), ent)
	return
}

func (a *Enveloping) Sync() error {
	return a.primary.Sync()
}
