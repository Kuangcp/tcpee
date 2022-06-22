package log

import (
	"io"
	"os"

	"codeberg.org/gruf/go-kv"
	"codeberg.org/gruf/go-logger/v2"
	"codeberg.org/gruf/go-logger/v2/level"
)

// Logger is the global logger instance.
var Logger = logger.NewWith(
	os.Stderr,
	logger.Config{Calldepth: 1},
	level.ALL,
	logger.Flags(0).SetTime(),
)

func Debug(a ...interface{}) {
	Logger.Debug(a...)
}

func Debugf(s string, a ...interface{}) {
	Logger.Debugf(s, a...)
}

func DebugKVs(fields ...kv.Field) {
	Logger.DebugKVs(fields...)
}

func Info(a ...interface{}) {
	Logger.Info(a...)
}

func Infof(s string, a ...interface{}) {
	Logger.Infof(s, a...)
}

func InfoKVs(fields ...kv.Field) {
	Logger.InfoKVs(fields...)
}

func Warn(a ...interface{}) {
	Logger.Warn(a...)
}

func Warnf(s string, a ...interface{}) {
	Logger.Warnf(s, a...)
}

func WarnKVs(fields ...kv.Field) {
	Logger.WarnKVs(fields...)
}

func Error(a ...interface{}) {
	Logger.Error(a...)
}

func Errorf(s string, a ...interface{}) {
	Logger.Errorf(s, a...)
}

func ErrorKVs(fields ...kv.Field) {
	Logger.ErrorKVs(fields...)
}

func Fatal(a ...interface{}) {
	Logger.Fatal(a...)
}

func Fatalf(s string, a ...interface{}) {
	Logger.Fatalf(s, a...)
}

func FatalKVs(fields ...kv.Field) {
	Logger.FatalKVs(fields...)
}

func Panic(a ...interface{}) {
	Logger.Panic(a...)
}

func Panicf(s string, a ...interface{}) {
	Logger.Panicf(s, a...)
}

func PanicKVs(fields ...kv.Field) {
	Logger.PanicKVs(fields...)
}

func Print(a ...interface{}) {
	Logger.Print(a...)
}

func Printf(s string, a ...interface{}) {
	Logger.Printf(s, a...)
}

func PrintKVs(fields ...kv.Field) {
	Logger.PrintKVs(fields...)
}

func Log(calldepth int, lvl level.LEVEL, a ...interface{}) {
	Logger.Log(calldepth, lvl, a...)
}

func Logf(calldepth int, lvl level.LEVEL, s string, a ...interface{}) {
	Logger.Logf(calldepth, lvl, s, a...)
}

func LogKVs(calldepth int, lvl level.LEVEL, fields ...kv.Field) {
	Logger.LogKVs(calldepth, lvl, fields...)
}

func Level() level.LEVEL {
	return Logger.Level()
}

func SetLevel(lvl level.LEVEL) {
	Logger.SetLevel(lvl)
}

func Flags() logger.Flags {
	return Logger.Flags()
}

func SetFlags(flags logger.Flags) {
	Logger.SetFlags(flags)
}

func SetOutput(out io.Writer) {
	Logger.SetOutput(out)
}

func Writer() io.Writer {
	return Logger.Writer()
}
