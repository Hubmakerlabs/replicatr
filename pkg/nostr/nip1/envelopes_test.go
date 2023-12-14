package nip1_test

import (
	"encoding/json"
	log2 "mleku.online/git/log"
	"mleku.online/git/mangle"
	"mleku.online/git/replicatr/pkg/nostr/nip1"
	"testing"
)

func TestEnveloper(t *testing.T) {
	log2.SetLogLevel(log2.Debug)
	const sub = "subscription000001"
	envs := []nip1.Enveloper{
		&nip1.EventEnvelope{SubscriptionID: sub, Event: events[0]},
		&nip1.EventEnvelope{Event: events[0]},
		&nip1.OKEnvelope{EventID: events[0].ID, OK: true,
			Reason: nip1.OKPoW + ": 25>24"},
		&nip1.ReqEnvelope{SubscriptionID: sub, Filters: filt},
		&nip1.NoticeEnvelope{Text: "this notice has been noticed"},
		&nip1.EOSEEnvelope{SubscriptionID: sub},
		&nip1.CloseEnvelope{SubscriptionID: sub},
	}
	var e error
	var b []byte
	for i := range envs {
		b, e = json.Marshal(envs[i])
		if e != nil {
			log.F.Ln(e)
			t.FailNow()
		}
		log.D.Ln("marshal  ", string(b))
		var env nip1.Enveloper
		var label []byte
		var buf *mangle.Buffer
		env, label, buf, e = nip1.ProcessEnvelope(b)
		if e != nil {
			log.F.Ln(e)
			t.FailNow()
		}
		log.D.Ln("unmarshal", env.ToArray().String())
		_ = env
		_ = label
		_ = buf
	}
}
