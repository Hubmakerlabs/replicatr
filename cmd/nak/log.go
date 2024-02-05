package main

import (
	"os"

	"mleku.online/git/slog"
)

var log = slog.New(os.Stderr)
