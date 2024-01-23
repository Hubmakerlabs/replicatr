package main

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

var log = slog.New(os.Stderr, "")
