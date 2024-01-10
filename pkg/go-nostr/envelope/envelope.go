package envelope

import (
	"bytes"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/OK"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/closed"
	close2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/closer"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/count"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/req"
)

func ParseMessage(message []byte) envelopes.E {
	firstComma := bytes.Index(message, []byte{','})
	if firstComma == -1 {
		return nil
	}
	label := message[0:firstComma]

	var v envelopes.E
	switch {
	case bytes.Contains(label, []byte("EVENT")):
		v = &event.E{}
	case bytes.Contains(label, []byte("REQ")):
		v = &req.Envelope{}
	case bytes.Contains(label, []byte("COUNT")):
		v = &count.Envelope{}
	case bytes.Contains(label, []byte("NOTICE")):
		x := notice.Envelope("")
		v = &x
	case bytes.Contains(label, []byte("EOSE")):
		x := eose.Envelope("")
		v = &x
	case bytes.Contains(label, []byte("OK")):
		v = &OK.Envelope{}
	case bytes.Contains(label, []byte("AUTH")):
		v = &auth.Envelope{}
	case bytes.Contains(label, []byte("CLOSED")):
		v = &closed.Envelope{}
	case bytes.Contains(label, []byte("CLOSE")):
		x := close2.Envelope("")
		v = &x
	default:
		return nil
	}

	if e := v.UnmarshalJSON(message); e != nil {
		return nil
	}
	return v
}
