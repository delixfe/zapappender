# zapappender

Composable appender for uber-go/zap enabling:

* Async logging
* Fallback
* Message Enveloping (like syslog formatting)

This project was created to allow logging to syslog over TCP.

## Quick start

Firstly, compose the appender chain:

```go
primaryOut := zapappender.NewWriter(zapcore.Lock(someTcpWriter))
consoleWriter := zapappender.NewWriter(zapcore.Lock(os.Stdout))
secondaryOut := zapappender.NewEnvelopingPreSuffix(consoleWriter, "FALLBACK: ", "")
fallback := zapappender.NewFallback(primaryOut, secondaryOut)
async, _ := zapappender.NewAsync(fallback,
    zapappender.AsyncOnQueueNearlyFullForwardTo(secondaryOut),
    zapappender.AsyncMaxQueueLength(10),
    zapappender.AsyncQueueMinFreePercent(0.2),
    zapappender.AsyncQueueMonitorPeriod(time.Millisecond),
)
appenderChain := async
```

Secondly, use that chain to create a `zapcore.Core` and finally to construct a `zap.Logger`.

```go
encoder := zapcore.NewConsoleEncoder(encoderConfig)
core := zapappender.NewAppenderCore(encoder, appenderChain, zapcore.DebugLevel)
logger := zap.New(core)

logger.Info("this logs async")
```

See [example_test.go](example_test.go) for more details.
