package logger

import (
	"fmt"
	"log"
	"os"
	"time"
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

// Logger 日志记录器
type Logger struct {
	level LogLevel
	file  *os.File
}

// NewLogger 创建新的日志记录器
func NewLogger(level string) *Logger {
	logLevel := InfoLevel
	switch level {
	case "debug":
		logLevel = DebugLevel
	case "info":
		logLevel = InfoLevel
	case "warn":
		logLevel = WarnLevel
	case "error":
		logLevel = ErrorLevel
	case "fatal":
		logLevel = FatalLevel
	}

	return &Logger{
		level: logLevel,
	}
}

// format 格式化日志消息
func (l *Logger) format(level string, msg string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s] [%s] %s", timestamp, level, fmt.Sprintf(msg, args...))
}

// Debug 记录调试级别日志
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level <= DebugLevel {
		log.Println(l.format("DEBUG", msg, args...))
	}
}

// Info 记录信息级别日志
func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level <= InfoLevel {
		log.Println(l.format("INFO", msg, args...))
	}
}

// Warn 记录警告级别日志
func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.level <= WarnLevel {
		log.Println(l.format("WARN", msg, args...))
	}
}

// Error 记录错误级别日志
func (l *Logger) Error(msg string, args ...interface{}) {
	if l.level <= ErrorLevel {
		log.Println(l.format("ERROR", msg, args...))
	}
}

// Fatal 记录致命级别日志并退出
func (l *Logger) Fatal(msg string, args ...interface{}) {
	log.Fatalln(l.format("FATAL", msg, args...))
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// WithFile 设置日志文件
func (l *Logger) WithFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	l.file = file
	return nil
}
