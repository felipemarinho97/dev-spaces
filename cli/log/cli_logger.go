package log

import (
	"fmt"
	"os"

	"github.com/felipemarinho97/dev-spaces/cli/util"
	"github.com/felipemarinho97/dev-spaces/core/log"
)

type cliLogger struct {
	ub    *util.UnknownBar
	level log.LogLevel
}

// NewCLILogger creates a new CLI logger implementation
func NewCLILogger() log.Logger {
	ub := util.NewUnknownBar("---")
	level := log.InfoLevel
	if logDebug() {
		level = log.DebugLevel
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
	if l.level <= log.DebugLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Info(args ...interface{}) {
	if l.level <= log.InfoLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Warn(args ...interface{}) {
	if l.level <= log.WarnLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Error(args ...interface{}) {
	if l.level <= log.ErrorLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Fatal(args ...interface{}) {
	if l.level <= log.FatalLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}

func (l *cliLogger) Panic(args ...interface{}) {
	if l.level <= log.PanicLevel {
		l.ub.SetDescription(fmt.Sprint(args...))
	}
}
