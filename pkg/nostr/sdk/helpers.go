package sdk

import (
	"os"

	"mleku.online/git/slog"
)

var log, chk = slog.New(os.Stderr)
