package log_test

import (
	"errors"
	"testing"

	l "github.com/Hubmakerlabs/replicatr/pkg/log"
)

var log = l.GetStd()

func TestGetLogger(t *testing.T) {
	l.SetLogLevel(l.Trace)
	log.T.Ln("testing log level", l.LvlStr[l.Trace])
	log.D.Ln("testing log level", l.LvlStr[l.Debug])
	log.I.Ln("testing log level", l.LvlStr[l.Info])
	log.W.Ln("testing log level", l.LvlStr[l.Warn])
	log.E.Ln("testing log level", l.LvlStr[l.Error])
	log.F.Ln("testing log level", l.LvlStr[l.Fatal])
	log.Fail(errors.New("dummy error as error"))
	log.I.Chk(errors.New("dummy information check"))
	log.I.Chk(nil)
	log.I.S("`backtick wrapped string`", t)
}
