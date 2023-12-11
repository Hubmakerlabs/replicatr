package nip1

import (
	"encoding/json"
	log2 "mleku.online/git/log"
	"mleku.online/git/mangle"
	"testing"
)

func TestEnveloper(t *testing.T) {
	log2.SetLogLevel(log2.Debug)
	const sub = "subscription000001"
	envs := []Enveloper{
		&EventEnvelope{sub, events[0]},
		&EventEnvelope{"", events[0]},
		&OKEnvelope{events[0].ID, true, OKPoW + ": 25>24"},
		&ReqEnvelope{sub, filt},
		&NoticeEnvelope{"this notice has been noticed"},
		&EOSEEnvelope{sub},
		&CloseEnvelope{sub},
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
		var env Enveloper
		var label []byte
		var buf *mangle.Buffer
		env, label, buf, e = ProcessEnvelope(b)
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
