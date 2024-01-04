package eventstore

import (
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
)

func GetAddrTagElements(tagValue string) (kind uint16, pkb []byte, d string) {
	spl := strings.Split(tagValue, ":")
	if len(spl) == 3 {
		if pkb, _ := hex.DecodeString(spl[1]); len(pkb) == 32 {
			if kind, e := strconv.ParseUint(spl[0], 10, 16); e == nil {
				return uint16(kind), pkb, spl[2]
			}
		}
	}
	return 0, nil, ""
}

func TagSorter(a, b tags.Tag) int {
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
