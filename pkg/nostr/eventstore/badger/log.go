package badger

import (
	"fmt"
	"strings"

	"mleku.dev/git/slog"
)

type logger struct {
	Level int
	Label string
}

func (l logger) Errorf(s string, i ...interface{}) {
	if l.Level >= slog.Error {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		log.E.Ln(strings.TrimSpace(txt))
	}
}

func (l logger) Warningf(s string, i ...interface{}) {
	if l.Level >= slog.Warn {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		log.W.F(strings.TrimSpace(txt))
	}
}

func (l logger) Infof(s string, i ...interface{}) {
	if l.Level >= slog.Info {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		log.I.F(strings.TrimSpace(txt))
	}
}

func (l logger) Debugf(s string, i ...interface{}) {
	if l.Level >= slog.Debug {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		log.D.F(strings.TrimSpace(txt))
	}
}
