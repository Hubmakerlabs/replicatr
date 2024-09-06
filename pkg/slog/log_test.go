package slog_test

import (
	"errors"
	"os"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

var log, chk = slog.New(os.Stdout)

func TestGetLogger(t *testing.T) {
	for i := 0; i < 100; i++ {
		slog.SetLogLevel(slog.Trace)
		log.T.Ln("testing log level", slog.LevelSpecs[slog.Trace].Name)
		log.D.Ln("testing log level", slog.LevelSpecs[slog.Debug].Name)
		log.I.Ln("testing log level", slog.LevelSpecs[slog.Info].Name)
		log.W.Ln("testing log level", slog.LevelSpecs[slog.Warn].Name)
		log.E.F("testing log level %s", slog.LevelSpecs[slog.Error].Name)
		log.F.Ln("testing log level", slog.LevelSpecs[slog.Fatal].Name)
		chk.F(errors.New("dummy error as fatal"))
		chk.E(errors.New("dummy error as error"))
		chk.W(errors.New("dummy error as warning"))
		chk.I(errors.New("dummy error as info"))
		chk.D(errors.New("dummy error as debug"))
		chk.T(errors.New("dummy error as trace"))
		log.I.Ln("log.I.Err",
			log.I.Err("format string %d '%s'", 5, "testing") != nil)
		log.I.Chk(errors.New("dummy information check"))
		log.I.Chk(nil)
		log.I.S("`backtick wrapped string`", t)
	}
}
