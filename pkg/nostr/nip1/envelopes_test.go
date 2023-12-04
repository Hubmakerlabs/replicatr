package nip1

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEventEnvelope(t *testing.T) {
	const sub = "subscription000001"
	envs := []Enveloper{
		&EventEnvelope{sub, events[0]},
		&OKEnvelope{events[0].ID, true, OKPOW + ": 25>24"},
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
			t.Fatal(e)
		}
		t.Log(string(b))
		var env Enveloper
		var label []byte
		var buf *bytes.Buffer
		env, label, buf, e = ProcessEnvelope(b)
		_ = env
		_ = label
		_ = buf
	}
}
