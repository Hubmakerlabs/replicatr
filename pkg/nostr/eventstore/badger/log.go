package badger

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/atomic"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

func NewLogger(logLevel int, label string) (l *logger) {
	log.I.Ln("getting logger for", label)
	l = &logger{Label: label}
	l.Level.Store(int32(logLevel))
	return
}

type logger struct {
	Level atomic.Int32
	Label string
}

func (l *logger) SetLogLevel(level int) {
	l.Level.Store(int32(level))
}

func (l *logger) Errorf(s string, i ...interface{}) {
	if l.Level.Load() >= slog.Error {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		_, file, line, _ := runtime.Caller(2)
		log.E.F("%s %s:%d", strings.TrimSpace(txt), file, line)
	}
}

func (l *logger) Warningf(s string, i ...interface{}) {
	if l.Level.Load() >= slog.Warn {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		_, file, line, _ := runtime.Caller(2)
		log.W.F("%s %s:%d", strings.TrimSpace(txt), file, line)
	}
}

func (l *logger) Infof(s string, i ...interface{}) {
	if l.Level.Load() >= slog.Info {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		_, file, line, _ := runtime.Caller(2)
		log.I.F("%s %s:%d", strings.TrimSpace(txt), file, line)
	}
}

func (l *logger) Debugf(s string, i ...interface{}) {
	if l.Level.Load() >= slog.Debug {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		_, file, line, _ := runtime.Caller(2)
		log.D.F("%s %s:%d", strings.TrimSpace(txt), file, line)
	}
}
