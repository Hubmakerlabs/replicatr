package badger

import (
	"fmt"
	"strings"

	"mleku.dev/git/slog"
)

type logger int

func (l logger) Errorf(s string, i ...interface{}) {
	if l <= slog.Error {
		s = "BADGER: " + s
		txt := fmt.Sprintf(s, i...)
		log.E.Ln(strings.TrimSpace(txt))
	}
}

func (l logger) Warningf(s string, i ...interface{}) {
	if l <= slog.Warn {
		s = "BADGER: " + s
		txt := fmt.Sprintf(s, i...)
		log.W.F(strings.TrimSpace(txt))
	}
}

func (l logger) Infof(s string, i ...interface{}) {
	if l <= slog.Info {
		s = "BADGER: " + s
		txt := fmt.Sprintf(s, i...)
		log.I.F(strings.TrimSpace(txt))
	}
}

func (l logger) Debugf(s string, i ...interface{}) {
	if l <= slog.Debug {
		s = "BADGER: " + s
		txt := fmt.Sprintf(s, i...)
		log.D.F(strings.TrimSpace(txt))
	}
}
