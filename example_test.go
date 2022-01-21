package zapappender_test

import (
	"context"
	"github.com/delixfe/zapappender"
	"os"
	"time"

	"github.com/delixfe/zapappender/chaos"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var encoderConfig = zapcore.EncoderConfig{
	MessageKey:       "msg",
	LevelKey:         "level",
	NameKey:          "logger",
	EncodeLevel:      zapcore.LowercaseLevelEncoder,
	EncodeTime:       zapcore.ISO8601TimeEncoder,
	EncodeDuration:   zapcore.StringDurationEncoder,
	ConsoleSeparator: " ** ",
}

func Example_core() {

	writer := zapappender.NewWriter(zapcore.Lock(os.Stdout))

	failing := chaos.NewFailingSwitchable(writer)

	// this could be a TcpWriter
	var primaryOut zapappender.Appender = failing

	// this would normally be os.Stdout or Stderr without further wrapping
	secondaryOut := zapappender.NewEnvelopingPreSuffix(writer, "FALLBACK: ", "")

	fallback := zapappender.NewFallback(primaryOut, secondaryOut)

	core := zapappender.NewAppenderCore(zapcore.NewConsoleEncoder(encoderConfig), fallback, zapcore.DebugLevel)
	logger := zap.New(core)

	logger.Info("zappig")

	failing.Break()

	logger.Info("on the fallback")

	// Output:
	// info ** zappig
	// FALLBACK: info ** on the fallback
}

func ExampleAsync() {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	writer := zapappender.NewWriter(zapcore.Lock(os.Stdout))

	failing := chaos.NewFailingSwitchable(writer)
	blocking := chaos.NewBlockingSwitchableCtx(ctx, failing)

	// this could be a TcpWriter
	var primaryOut zapappender.Appender = zapappender.NewEnvelopingPreSuffix(blocking, "PRIMARY:  ", "")

	// this would normally be os.Stdout or Stderr without further wrapping
	secondaryOut := zapappender.NewEnvelopingPreSuffix(writer, "FALLBACK: ", "")

	fallback := zapappender.NewFallback(primaryOut, secondaryOut)
	async, _ := zapappender.NewAsync(fallback,
		zapappender.AsyncOnQueueNearlyFullForwardTo(zapappender.NewEnvelopingPreSuffix(writer, "QFALLBACK: ", "")),
		zapappender.AsyncMaxQueueLength(10),
		zapappender.AsyncQueueMinFreePercent(0.2),
		zapappender.AsyncQueueMonitorPeriod(time.Millisecond),
	)

	core := zapappender.NewAppenderCore(zapcore.NewConsoleEncoder(encoderConfig), async, zapcore.DebugLevel)
	logger := zap.New(core)

	logger.Info("this logs async")

	time.Sleep(time.Millisecond * 10)

	blocking.Break()

	logger.Info("primary blocks while trying to send this", zap.Int("i", 1))
	for i := 2; i <= 15; i++ {
		logger.Info("while broken", zap.Int("i", i))
	}

	blocking.Fix()
	time.Sleep(time.Millisecond * 10)
	async.Drain(ctx)

	// Output:
	// PRIMARY:  info ** this logs async
	// QFALLBACK: info ** while broken ** {"i": 2}
	// QFALLBACK: info ** while broken ** {"i": 3}
	// QFALLBACK: info ** while broken ** {"i": 4}
	// QFALLBACK: info ** while broken ** {"i": 5}
	// PRIMARY:  info ** primary blocks while trying to send this ** {"i": 1}
	// PRIMARY:  info ** while broken ** {"i": 6}
	// PRIMARY:  info ** while broken ** {"i": 7}
	// PRIMARY:  info ** while broken ** {"i": 8}
	// PRIMARY:  info ** while broken ** {"i": 9}
	// PRIMARY:  info ** while broken ** {"i": 10}
	// PRIMARY:  info ** while broken ** {"i": 11}
	// PRIMARY:  info ** while broken ** {"i": 12}
	// PRIMARY:  info ** while broken ** {"i": 13}
	// PRIMARY:  info ** while broken ** {"i": 14}
	// PRIMARY:  info ** while broken ** {"i": 15}
}
