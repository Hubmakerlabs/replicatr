package event_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/ec/secp256k1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event/eventest"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/quotes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
	"github.com/davecgh/go-spew/spew"
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
	TestEvents = []string{
		`{"id":"ed89902da916be56f598af0114384d4f878fa17612d99590fb848d74def7e0b7","pubkey":"8fe53b37518e3dbe9bab26d912292001d8b882de9456b7b08b615f912dc8bf4a","created_at":1709853586,"kind":9735,"tags":[["p","22f7161f76e075b9e0a250a447884ac09b04b636effd7c703a92394ed3fb39e8"],["e","d47916eec6fc51aae8f7d5792cbb2f7a4eca4cddc1a91b5f46460799c1345486"],["bolt11","lnbc210n1pj75judpp5gcag9vwctv2snc08vxa4xxegpvjyjhfn0mn9wg7sj39866vgswvshp5ac0xy9vx9csn5h0s8vjv8klvptsvpuspgg60zf9gwn6tyz9cmfescqzpuxqyz5vqsp5ya8v9arwyep46s9pdlxejefrrcp5qg36vtksqyu3q59hav32ekrq9qyyssq0s8pfn8pu5c8g8kg3rxa650nthdknhka228y2q6ud52d3g6nhv3h5kwt4k7mzegvl7mt3g57lyusgk4t0yy5mnhywlulxyl5jlsx7usp0rc3g0"],["description","{\"id\":\"e5d361a724caaa94af693d2892d6a30237d7281039085a2c9fdec711f8dfd498\",\"pubkey\":\"af387a6c488c5484088ba715dbb42b55ce72b475e1e2b86be791b24b8d51e215\",\"created_at\":1709853580,\"kind\":9734,\"tags\":[[\"p\",\"22f7161f76e075b9e0a250a447884ac09b04b636effd7c703a92394ed3fb39e8\"],[\"e\",\"d47916eec6fc51aae8f7d5792cbb2f7a4eca4cddc1a91b5f46460799c1345486\"],[\"amount\",\"21000\"],[\"relays\",\"wss://relay.nostr.info\",\"wss://nostr.zebedee.cloud\",\"wss://nostr.orangepill.dev\",\"wss://nostr.bitcoiner.social\",\"wss://relay.damus.io\",\"wss://nostr-relay.wlvs.space\",\"wss://relay.snort.social\",\"wss://nostr-pub.semisol.dev\",\"wss://relay.current.fyi\",\"wss://eden.nostr.land\",\"wss://brb.io\"]],\"content\":\"Onward 🫡\",\"sig\":\"cfb67577444da108b1cca6866af1fe007bdad25b009deea9641a98e47b55bda799a31e4f372ed3c82c25181b7e56d866c2adb6d9305bb827982b1aa40b036260\"}"],["preimage","6ada9db8ff695cc22fea7248c2f7f46f83b06c9485db06d26ad0dbbef726d141"]],"content":"{\"key\":\"value\"}","sig":"c3c35e4f39c03711ace2f52122a6b7717f916e94725e5a3ff738525fc876c7ba17bc85470c7365e8997421202818fab26a5356eac59a1bc545292558d1ca7efb"}`,
	}
)

func TestEscaping(t *testing.T) {
	for _, evt := range TestEvents {
		t.Log(evt)
		ev := &event.T{}
		err := json.Unmarshal([]byte(evt), ev)
		if err != nil {
			t.Error(err)
		}
		t.Log(spew.Sdump(ev))
		t.Log(ev.ToObject().String())
		var j []byte
		if j, err = json.Marshal(ev); chk.E(err) {
			t.Error(err)
		}

		t.Log(string(j))
	}
}

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
