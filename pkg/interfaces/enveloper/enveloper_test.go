package enveloper_test

import (
	"encoding/json"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/interfaces/enveloper"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes"
	close2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eoseenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/noticeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventest"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filtertest"
)

var log = log2.GetStd()

func TestEnveloper(t *testing.T) {
	log2.SetLogLevel(log2.Debug)
	const sub = "subscription000001"
	envs := []enveloper.I{
		&eventenvelope.T{SubscriptionID: sub, Event: eventest.D[0]},
		&eventenvelope.T{SubscriptionID: sub, Event: eventest.D[1]},
		&eventenvelope.T{Event: eventest.D[0]},
		&okenvelope.T{EventID: eventest.D[0].ID, OK: true,
			Reason: okenvelope.Message(okenvelope.PoW, "25>24 \\ ")},
		&reqenvelope.T{SubscriptionID: sub, T: filtertest.D},
		&noticeenvelope.T{Text: "this notice has been noticed } \\ \\\" ] "},
		&eoseenvelope.T{T: sub},
		&close2.T{T: sub},
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
		env, _, e = envelopes.ProcessEnvelope(b)
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
