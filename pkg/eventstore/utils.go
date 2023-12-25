package eventstore

import (
	"encoding/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"strconv"
	"strings"
)

func GetAddrTagElements(tagValue string) (k kind.T, pkb []byte, d string) {
	spl := strings.Split(tagValue, ":")
	if len(spl) == 3 {
		if pkb, _ := hex.DecodeString(spl[1]); len(pkb) == 32 {
			if k, err := strconv.ParseUint(spl[0], 10, 16); err == nil {
				return kind.T(k), pkb, spl[2]
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
