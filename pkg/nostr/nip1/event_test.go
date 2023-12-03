package nip1

import (
	"encoding/json"
	"mleku.online/git/replicatr/pkg/nostr/kind"
	"mleku.online/git/replicatr/pkg/nostr/tag"
	"mleku.online/git/replicatr/pkg/nostr/tags"
	"mleku.online/git/replicatr/pkg/nostr/timestamp"
	"testing"
)

var events = []*Event{
	{
		ID:        "92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
		PubKey:    "e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
		Kind:      kind.EncryptedDirectMessage,
		CreatedAt: timestamp.T(1671028682),
		Tags: tags.T{tag.T{
			"p",
			"f8340b2bde651576b75af61aa26c80e13c65029f00f7f64004eece679bf7059f",
		}},
		Content: "you say yes, I say no",
		Sig: "ed08d2dd5b0f7b6a3cdc74643d4adee3158ddede9cc848e8cd97630c097001ac" +
			"c2d052d2d3ec2b7ac4708b2314b797106d1b3c107322e61b5e5cc2116e099b79",
	},
}

func TestEventSerialization(t *testing.T) {
	for _, evt := range events {

		var b []byte
		var e error

		b, e = json.Marshal(evt)
		t.Log(string(b))
		var re Event
		if e = json.Unmarshal(b, &re); e != nil {
			t.Log(string(b))
			t.Error("failed to re parse event just serialized", e)
		}

		if evt.ID != re.ID || evt.PubKey != re.PubKey || evt.Content != re.Content ||
			evt.CreatedAt != re.CreatedAt || evt.Sig != re.Sig ||
			len(evt.Tags) != len(re.Tags) {
			t.Error("reparsed event differs from original")
		}

		for i := range evt.Tags {
			if len(evt.Tags[i]) != len(re.Tags[i]) {
				t.Errorf("reparsed tags %d length differ from original", i)
				continue
			}

			for j := range evt.Tags[i] {
				if evt.Tags[i][j] != re.Tags[i][j] {
					t.Errorf("reparsed tag content %d %d length differ from original",
						i, j)
				}
			}
		}
	}
}
