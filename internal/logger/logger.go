package logger

import (
	"log"

	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	localLogger, err := zap.NewDevelopment()

	if err != nil {
		log.Fatalf("Error intializing logger: %v", err)
	}

	logger = localLogger
}

func Fatal(msg string, keysAndValues ...any) {
	sugar := logger.Sugar()
	sugar.Fatalw(msg, keysAndValues...)
}

func Error(msg string, keysAndValues ...any) {
	sugar := logger.Sugar()
	sugar.Errorw(msg, keysAndValues...)
}

func Warning(msg string, keysAndValues ...any) {
	sugar := logger.Sugar()
	sugar.Warnw(msg, keysAndValues...)
}

func Info(msg string, keysAndValues ...any) {
	sugar := logger.Sugar()
	sugar.Infow(msg, keysAndValues...)
}

func Debug(msg string, keysAndValue ...any) {
	sugar := logger.Sugar()
	sugar.Debugw(msg, keysAndValue...)
}

func DebugZap(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}
