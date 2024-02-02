package kinds

import "github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"

var PrivilegedKinds = T{
	kind.EncryptedDirectMessage,
	kind.GiftWrap,
	kind.GiftWrapWithKind4,
	kind.ApplicationSpecificData,
}

func IsPrivileged(k kind.T) (is bool) {
	for i := range PrivilegedKinds {
		if k == PrivilegedKinds[i] {
			return true
		}
	}
	return
}
