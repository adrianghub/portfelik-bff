package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
		debugLogger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.infoLogger.Output(2, fmt.Sprintf(format, v...))
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.errorLogger.Output(2, fmt.Sprintf(format, v...))
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.debugLogger.Output(2, fmt.Sprintf(format, v...))
}

func (l *Logger) LogRequest(method, path, remoteAddr string, duration time.Duration) {
	l.infoLogger.Printf(
		"Request: %s %s %s - %s",
		method,
		path,
		remoteAddr,
		duration,
	)
}
