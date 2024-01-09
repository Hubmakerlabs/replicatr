package enveloper_test

import (
	"encoding/json"
	"testing"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/OK"
	close2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/req"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventest"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filtertest"
)

var log = log2.GetStd()

func TestEnveloper(t *testing.T) {
	// log2.SetLogLevel(log2.Debug)
	const sub = "subscription000001"
	envs := []enveloper.I{
		&event.Envelope{SubscriptionID: sub, Event: eventest.D[0]},
		&event.Envelope{SubscriptionID: sub, Event: eventest.D[1]},
		&event.Envelope{Event: eventest.D[0]},
		&OK.Envelope{EventID: eventest.D[0].ID, OK: true,
			Reason: OK.Message(OK.PoW, "25>24 \\ ")},
		&req.Envelope{SubscriptionID: sub, T: filtertest.D},
		&notice.Envelope{Text: "this notice has been noticed } \\ \\\" ] "},
		&eose.Envelope{T: sub},
		&close2.Envelope{T: sub},
	}
	var e error
	var b []byte
	for i := range envs {
		b, e = json.Marshal(envs[i])
		if e != nil {
			t.Fatal(e)
		}
		marshaled := string(b)
		log.D.Ln("marshaled  ", marshaled)
		var env enveloper.I
		env, _, _, e = envelopes.ProcessEnvelope(b)
		if e != nil {
			t.Fatal(e)
		}
		var um []byte
		log.I.Ln("marshaling")
		um, e = json.Marshal(env)
		unmarshaled := string(um)
		log.D.Ln("unmarshaled", unmarshaled)
		if marshaled != unmarshaled {
			t.Log("marshal/unmarshal mangled.")
			t.Log("got:     ", unmarshaled)
			t.Log("expected:", marshaled)
			t.FailNow()
		}
	}
}
