package eventstore

import (
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

func GetAddrTagElements(tagValue string) (k uint16, pkb []byte, d string) {
	spl := strings.Split(tagValue, ":")
	if len(spl) == 3 {
		if pkb, _ = hex.DecodeString(spl[1]); len(pkb) == 32 {
			if kk, e := strconv.ParseUint(spl[0], 10, 16); log.E.Chk(e) {
				return uint16(kk), pkb, spl[2]
			}
		}
	}
	return 0, nil, ""
}

func TagSorter(a, b tag.T) int {
	if len(a) < 2 {
		if len(b) < 2 {
			return 0
		}
		return -1
	}
	if len(b) < 2 {
		return 1
	}
	return strings.Compare(a[1], b[1])
}
