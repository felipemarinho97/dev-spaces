package log

import (
	"fmt"
	"os"

	"github.com/felipemarinho97/dev-spaces/util"
)

type cliLogger struct {
	ub    *util.UnknownBar
	level LogLevel
}

func NewCLILogger() Logger {
	ub := util.NewUnknownBar("---")
	level := InfoLevel
	if logDebug() {
		level = DebugLevel
	}
	return &cliLogger{
		ub:    ub,
		level: level,
	}
}

func logDebug() bool {
	return os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1"
}

func (l *cliLogger) Debug(args ...interface{}) {
	if l.level <= DebugLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Info(args ...interface{}) {
	if l.level <= InfoLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Warn(args ...interface{}) {
	if l.level <= WarnLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Error(args ...interface{}) {
	if l.level <= ErrorLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Fatal(args ...interface{}) {
	if l.level <= FatalLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Panic(args ...interface{}) {
	if l.level <= PanicLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}
