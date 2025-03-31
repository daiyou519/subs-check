package logger

import (
	"bytes"
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

type LogLevel int

var LogLevelSet LogLevel

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
	LogLevelPanic
	InfoColor  = "\033[34m"
	WarnColor  = "\033[33m"
	ErrorColor = "\033[31m"
	DebugColor = "\033[32m"
	ResetColor = "\033[0m"
	TimeFormat = "2006-01-02T15:04:05"
)

func getCallerInfo() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "unknown:0"
	}

	workDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Sprintf("%s:%d", file, line)
	}

	relPath, err := filepath.Rel(workDir, file)
	if err != nil {
		return fmt.Sprintf("%s:%d", file, line)
	}

	return fmt.Sprintf("%s:%d", relPath, line)
}

func log(level LogLevel, format string, v ...any) {
	if level < LogLevelSet {
		return
	}

	var levelStr string
	var color string
	switch level {
	case LogLevelInfo:
		levelStr = "INFO"
		color = InfoColor
	case LogLevelWarn:
		levelStr = "WARN"
		color = WarnColor
	case LogLevelError:
		levelStr = "ERROR"
		color = ErrorColor
	case LogLevelFatal:
		levelStr = "FATAL"
		color = ErrorColor
	case LogLevelDebug:
		levelStr = "DEBUG"
		color = DebugColor
	case LogLevelPanic:
		levelStr = "PANIC"
		color = ErrorColor
	}

	var buf bytes.Buffer
	buf.WriteString(color)
	buf.WriteString(fmt.Sprintf("%-5s", levelStr))
	buf.WriteString(ResetColor)
	buf.WriteString(" [")
	buf.WriteString(time.Now().Format(TimeFormat))
	buf.WriteString("] [")
	buf.WriteString(getCallerInfo())
	buf.WriteString("] ")
	buf.WriteString(fmt.Sprintf(format, v...))
	buf.WriteByte('\n')

	fmt.Print(buf.String())
}
func SetLogLevel(level string) {
	switch level {
	case "debug":
		LogLevelSet = LogLevelDebug
	case "info":
		LogLevelSet = LogLevelInfo
	case "warn":
		LogLevelSet = LogLevelWarn
	case "error":
		LogLevelSet = LogLevelError
	case "fatal":
		LogLevelSet = LogLevelFatal
	case "panic":
		LogLevelSet = LogLevelPanic
	}
}
func Info(format string, v ...any) {
	log(LogLevelInfo, format, v...)
}

func Warn(format string, v ...any) {
	log(LogLevelWarn, format, v...)
}

func Error(format string, v ...any) {
	log(LogLevelError, format, v...)
}

func Fatal(format string, v ...any) {
	log(LogLevelFatal, format, v...)
}
func Debug(format string, v ...any) {
	log(LogLevelDebug, format, v...)
}
func Panic(format string, v ...any) {
	log(LogLevelPanic, format, v...)
}
