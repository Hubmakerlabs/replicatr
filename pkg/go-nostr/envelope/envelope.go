package envelope

import (
	"bytes"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/OK"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/auth"
	close2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/close"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/closed"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/count"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/req"
)

func ParseMessage(message []byte) envelopes.Envelope {
	firstComma := bytes.Index(message, []byte{','})
	if firstComma == -1 {
		return nil
	}
	label := message[0:firstComma]

	var v envelopes.Envelope
	switch {
	case bytes.Contains(label, []byte("EVENT")):
		v = &event.EventEnvelope{}
	case bytes.Contains(label, []byte("REQ")):
		v = &req.ReqEnvelope{}
	case bytes.Contains(label, []byte("COUNT")):
		v = &count.CountEnvelope{}
	case bytes.Contains(label, []byte("NOTICE")):
		x := notice.NoticeEnvelope("")
		v = &x
	case bytes.Contains(label, []byte("EOSE")):
		x := eose.EOSEEnvelope("")
		v = &x
	case bytes.Contains(label, []byte("OK")):
		v = &OK.OKEnvelope{}
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
