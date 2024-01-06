package sdk

import (
	"encoding/hex"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
)

var log, fails = log2.GetStd()

var (
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)
