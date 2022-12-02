package log

import (
	"io"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/vietanhduong/xcontroller/pkg/util/env"
)

var logger *zap.Logger

func init() {
	w := io.Writer(os.Stdout)
	logger = zap.New(zapcore.NewCore(DefaultEncoder(), zapcore.AddSync(w), parseLevel(env.StringFromEnv("LOG_LEVEL", "info"))), zap.AddCaller())
	zap.ReplaceGlobals(logger) // set as default logger
}

func NewLogger(level string) *zap.Logger {
	w := io.Writer(os.Stdout)
	logger = zap.New(zapcore.NewCore(DefaultEncoder(), zapcore.AddSync(w), parseLevel(level)), zap.AddCaller())
	return logger
}

func NewK8sLogger(level string) logr.Logger {
	return kzap.New(kzap.Level(parseLevel(level)), kzap.UseDevMode(true), kzap.WriteTo(os.Stdout), kzap.Encoder(DefaultEncoder()))
}

func DefaultEncoder() zapcore.Encoder {
	conf := zap.NewDevelopmentEncoderConfig()
	conf.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(conf)
	return encoder
}

func parseLevel(level string) zapcore.Level {
	l, err := zapcore.ParseLevel(strings.ToLower(level))
	if err != nil {
		return zap.InfoLevel
	}
	return l
}

func Info(args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Info(args...)
}

func Infof(format string, args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Infof(format, args...)
}

func Warn(args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Warnf(format, args...)
}

func Error(args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Error(args...)
}

func Errorf(format string, args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Errorf(format, args...)
}

func Debug(args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Debugf(format, args...)
}

func Fatal(args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Sugar().Fatalf(format, args...)
}
