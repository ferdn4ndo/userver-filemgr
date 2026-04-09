package lib

import (
	"strings"

	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

type GinLogger struct {
	*Logger
}

type FxLogger struct {
	*Logger
}

var (
	globalLogger *Logger
	zapLogger    *zap.Logger
)

func GetLogger() Logger {
	if globalLogger == nil {
		logger := newLogger(NewEnv())
		globalLogger = &logger
	}
	return *globalLogger
}

func (l *Logger) GetGinLogger() GinLogger {
	logger := zapLogger.WithOptions(zap.WithCaller(false))
	return GinLogger{Logger: newSugaredLogger(logger)}
}

func (l *Logger) GetFxLogger() fxevent.Logger {
	logger := zapLogger.WithOptions(zap.WithCaller(false))
	return &FxLogger{Logger: newSugaredLogger(logger)}
}

func (l *FxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.Debug("OnStart failed: ", zap.String("callee", e.FunctionName), zap.Error(e.Err))
		}
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.Debug("OnStop failed: ", zap.String("callee", e.FunctionName), zap.Error(e.Err))
		}
	case *fxevent.Started:
		if e.Err == nil {
			l.Debug("started")
		}
	}
}

func (l *FxLogger) Printf(str string, args ...interface{}) {
	if len(args) > 0 {
		l.Debugf(str, args)
		return
	}
	l.Debug(str)
}

func newSugaredLogger(logger *zap.Logger) *Logger {
	return &Logger{SugaredLogger: logger.Sugar()}
}

func newLogger(env Env) Logger {
	config := zap.NewProductionConfig()
	if env.IsLocal() {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	level := zapcore.InfoLevel
	switch env.LogLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}
	config.Level.SetLevel(level)
	zapLogger, _ = config.Build()
	return *newSugaredLogger(zapLogger)
}

func (l GinLogger) Write(p []byte) (n int, err error) {
	msg := strings.TrimRight(string(p), "\r\n")
	if msg != "" {
		l.Info(msg)
	}
	return len(p), nil
}
