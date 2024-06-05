package enveloper_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eoseenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/noticeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event/eventest"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters/filtertest"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"mleku.net/slog"
)

var log, chk = slog.New(os.Stderr)

func TestEnveloper(t *testing.T) {
	slog.SetLogLevel(slog.Debug)
	const sub = "subscription000001"
	envs := []enveloper.I{
		&eventenvelope.T{SubscriptionID: sub, Event: eventest.D[0]},
		&eventenvelope.T{SubscriptionID: sub, Event: eventest.D[1]},
		&eventenvelope.T{Event: eventest.D[0]},
		&okenvelope.T{ID: eventest.D[0].ID, OK: true,
			Reason: okenvelope.Message(okenvelope.PoW, "25>24 \\ ")},
		&reqenvelope.T{SubscriptionID: sub, Filters: filtertest.D},
		&noticeenvelope.T{Text: "this notice has been noticed } \\ \\\" ] "},
		&eoseenvelope.T{Sub: sub},
		&closeenvelope.T{T: sub},
	}
	var err error
	var b []byte
	for i := range envs {
		b, err = json.Marshal(envs[i])
		if err != nil {
			t.Fatal(err)
		}
		marshaled := string(b)
		log.D.Ln("marshaled   ", marshaled)
		var env enveloper.I
		env, _, err = envelopes.ProcessEnvelope(b)
		if err != nil {
			t.Fatal(err)
		}
		var um []byte
		log.I.Ln("marshaling")
		um, err = json.Marshal(env)
		unmarshaled := string(um)
		log.D.Ln("unmarshaled ", unmarshaled)
		if marshaled != unmarshaled {
			t.Log("marshal/unmarshal mangled.")
			t.Log("got:     ", unmarshaled)
			t.Log("expected:", marshaled)
			t.FailNow()
		}
	}
}
