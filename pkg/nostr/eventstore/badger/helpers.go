package badger

import (
	"os"

	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)