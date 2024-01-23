package slog_test

import (
	"errors"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

var log = slog.GetStd()

func TestGetLogger(t *testing.T) {
	slog.SetLogLevel(slog.Trace)
	log.T.Ln("testing log level", slog.LvlStr[slog.Trace])
	log.D.Ln("testing log level", slog.LvlStr[slog.Debug])
	log.I.Ln("testing log level", slog.LvlStr[slog.Info])
	log.W.Ln("testing log level", slog.LvlStr[slog.Warn])
	log.E.Ln("testing log level", slog.LvlStr[slog.Error])
	log.F.Ln("testing log level", slog.LvlStr[slog.Fatal])
	log.Fail(errors.New("dummy error as error"))
	log.I.Chk(errors.New("dummy information check"))
	log.I.Chk(nil)
	log.I.S("`backtick wrapped string`", t)
}
