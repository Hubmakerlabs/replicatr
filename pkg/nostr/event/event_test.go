package event_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event/eventest"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/quotes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	secp256k1 "mleku.dev/git/ec/secp256k1"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

const (
	TestSecBech32 = "nsec1z7tlduw3qkf4fz6kdw3jaq2h02jtexgwkrck244l3p834a930sjsh8t89c"
	TestPubBech32 = "npub1flds0h62dqlra6dj48g30cqmlcj534lgcr2vk4kh06wzxzgu8lpss5kaa2"
	TestSecHex    = "1797f6f1d10593548b566ba32e81577aa4bc990eb0f16556bf884f1af4b17c25"
	TestPubHex    = "4fdb07df4a683e3ee9b2a9d117e01bfe2548d7e8c0d4cb56d77e9c23091c3fc3"
)

func GetTestKeyPair() (sec *secp256k1.SecretKey,
	pub *secp256k1.PublicKey) {
	b, _ := hex.Dec(TestSecHex)
	sec = secp256k1.SecKeyFromBytes(b)
	pub = sec.PubKey()
	return
}

var (
	TestEventContent = []string{
		`This event contains { braces } and [ brackets ] that must be properly 
handled, as well as a line break, a dangling space and a 
	tab.`,
	}
)

func GenTextNote(sk *secp256k1.SecretKey, replyID,
	relayURL string) (note string, err error) {

	// pick random quote to use in content field of event
	src := rand.Intn(len(quotes.D))
	q := rand.Intn(len(quotes.D[src].Paragraphs))
	quoteText := fmt.Sprintf("\"%s\" - %s",
		quotes.D[src].Paragraphs[q],
		quotes.D[src].Source)
	var t tags.T
	tagMarker := tag.MarkerRoot
	if replyID != "" {
		tagMarker = tag.MarkerReply
	}
	t = tags.T{{"e", replyID, relayURL, tagMarker}}
	ev := &event.T{
		CreatedAt: timestamp.Now(),
		Kind:      kind.TextNote,
		Tags:      t,
		Content:   quoteText,
	}
	if err = ev.SignWithSecKey(sk); chk.D(err) {
		return
	}
	note = ev.ToObject().String()
	return
}

func TestGenerateEvent(t *testing.T) {
	// log2.SetLogLevel(log2.Debug)
	var err error
	var note, noteID, relayURL string
	sec, pub := GetTestKeyPair()
	_ = pub
	for i := 0; i < 10; i++ {
		if note, err = GenTextNote(sec, noteID, relayURL); chk.D(err) {
			t.Error(err)
			t.FailNow()
		}
		log.D.Ln(note)
	}
}

func TestEventSerialization(t *testing.T) {
	for _, evt := range eventest.D {

		var b []byte
		var err error

		b, err = json.Marshal(evt)
		// t.Log(string(b))
		var re event.T
		if err = json.Unmarshal(b, &re); err != nil {
			t.Log(string(b))
			t.Error("failed to re parse event just serialized", err)
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
