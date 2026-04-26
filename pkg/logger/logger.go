package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// LogLevel 日志级别
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// Logger 日志记录器（兼容原有格式化调用方式）
type Logger struct {
	level  LogLevel
	logger *slog.Logger
	file   *os.File
}

var (
	globalMu     sync.RWMutex
	globalLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
)

// Init 初始化全局日志器。
func Init(level string, format string) {
	handler := buildHandler(os.Stdout, level, format)

	globalMu.Lock()
	globalLogger = slog.New(handler)
	globalMu.Unlock()
}

// S 返回当前全局结构化日志器。
func S() *slog.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

// Component 返回带组件标签的日志器。
func Component(name string) *slog.Logger {
	return S().With("component", strings.TrimSpace(name))
}

// NewLogger 创建新的日志记录器。
func NewLogger(level string) *Logger {
	return &Logger{
		level:  parseLegacyLevel(level),
		logger: S(),
	}
}

func parseLegacyLevel(level string) LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return DebugLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

func parseSlogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func buildHandler(w io.Writer, level string, format string) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     parseSlogLevel(level),
		AddSource: true,
	}
	if strings.EqualFold(strings.TrimSpace(format), "text") {
		return slog.NewTextHandler(w, opts)
	}
	return slog.NewJSONHandler(w, opts)
}

func (l *Logger) logf(level slog.Level, threshold LogLevel, msg string, args ...interface{}) {
	if l == nil {
		return
	}
	if l.level > threshold {
		return
	}

	content := msg
	if len(args) > 0 {
		content = fmt.Sprintf(msg, args...)
	}

	base := l.logger
	if base == nil {
		base = S()
	}
	base.Log(nil, level, content)
}

// Debug 记录调试级别日志。
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.logf(slog.LevelDebug, DebugLevel, msg, args...)
}

// Info 记录信息级别日志。
func (l *Logger) Info(msg string, args ...interface{}) {
	l.logf(slog.LevelInfo, InfoLevel, msg, args...)
}

// Warn 记录警告级别日志。
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.logf(slog.LevelWarn, WarnLevel, msg, args...)
}

// Error 记录错误级别日志。
func (l *Logger) Error(msg string, args ...interface{}) {
	l.logf(slog.LevelError, ErrorLevel, msg, args...)
}

// Fatal 记录致命级别日志并退出。
func (l *Logger) Fatal(msg string, args ...interface{}) {
	content := msg
	if len(args) > 0 {
		content = fmt.Sprintf(msg, args...)
	}
	S().Error(content, slog.String("severity", "fatal"))
	os.Exit(1)
}

// Close 关闭日志文件。
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// WithFile 输出日志到指定文件（并同步到stdout）。
func (l *Logger) WithFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return err
	}

	level := "info"
	switch l.level {
	case DebugLevel:
		level = "debug"
	case WarnLevel:
		level = "warn"
	case ErrorLevel, FatalLevel:
		level = "error"
	}

	l.file = file
	l.logger = slog.New(buildHandler(io.MultiWriter(os.Stdout, file), level, "json"))
	return nil
}
