package jobq

import (
	"fmt"
	"log"
	"os"
)

// Logger interface is used throughout jobq.
type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Info(args ...any)
	Error(args ...any)
	Fatal(args ...any)
}

type defaultLogger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
}

func NewLogger() Logger {
	return &defaultLogger{
		infoLogger:  log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime),
		fatalLogger: log.New(os.Stderr, "FATAL: ", log.Ldate|log.Ltime),
	}
}

func (l *defaultLogger) logWithCallerf(logger *log.Logger, format string, args ...any) {
	// Assuming stack(3) returns caller info string
	stackInfo := stack(3)
	// Prepend stack info to the arguments and adjust the format string
	logger.Printf("%s "+format, append([]any{stackInfo}, args...)...)
}

func (l *defaultLogger) logWithCaller(logger *log.Logger, args ...any) {
	stack := stack(3)
	fmt.Println(append([]any{stack}, args...)...)
}

func (l *defaultLogger) Info(args ...any) {
	l.infoLogger.Println(fmt.Sprint(args...))
}

func (l *defaultLogger) Error(args ...any) {
	l.errorLogger.Println(fmt.Sprint(args...))
}

func (l *defaultLogger) Fatal(args ...any) {
	l.logWithCaller(l.fatalLogger, args...)
}

func (l *defaultLogger) Infof(format string, args ...any) {
	l.infoLogger.Printf(format, args...)
}

func (l *defaultLogger) Errorf(format string, args ...any) {
	l.errorLogger.Printf(format, args...)
}

func (l *defaultLogger) Fatalf(format string, args ...any) {
	l.logWithCallerf(l.fatalLogger, format, args...)
}

type emptyLogger struct{}

func NewEmptyLogger() Logger {
	return &emptyLogger{}
}

func (l emptyLogger) Infof(format string, args ...any)  {}
func (l emptyLogger) Errorf(format string, args ...any) {}
func (l emptyLogger) Fatalf(format string, args ...any) {}
func (l emptyLogger) Info(args ...any)                  {}
func (l emptyLogger) Error(args ...any)                 {}
func (l emptyLogger) Fatal(args ...any)                 {}
