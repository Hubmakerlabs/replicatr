package nip1_test

import (
	"encoding/json"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"testing"
)

func TestEnveloper(t *testing.T) {
	// log2.SetLogLevel(log2.Debug)
	const sub = "subscription000001"
	envs := []nip1.Enveloper{
		&nip1.EventEnvelope{SubscriptionID: sub, Event: events[0]},
		&nip1.EventEnvelope{SubscriptionID: sub, Event: events[1]},
		&nip1.EventEnvelope{Event: events[0]},
		&nip1.OKEnvelope{EventID: events[0].ID, OK: true,
			Reason: nip1.OKMessage(nip1.OKPoW, "25>24 \\ ")},
		&nip1.ReqEnvelope{SubscriptionID: sub, Filters: filt},
		&nip1.NoticeEnvelope{Text: "this notice has been noticed } \\ \\\" ] "},
		&nip1.EOSEEnvelope{SubscriptionID: sub},
		&nip1.CloseEnvelope{SubscriptionID: sub},
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
		var env nip1.Enveloper
		env, _, _, e = nip1.ProcessEnvelope(b)
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
