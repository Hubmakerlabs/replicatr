package sdk

import (
	"os"

	"mleku.net/slog"
)

var log, chk = slog.New(os.Stderr)
