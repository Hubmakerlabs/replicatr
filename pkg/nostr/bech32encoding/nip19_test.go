package bech32encoding

import (
	"reflect"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"
)

func TestEncodeNpub(t *testing.T) {
	npub, e := EncodePublicKey("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	if e != nil {
		t.Errorf("shouldn't error: %s", e)
	}
	if npub != "npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6" {
		t.Error("produced an unexpected npub string")
	}
}

func TestEncodeNsec(t *testing.T) {
	nsec, e := EncodePrivateKey("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	if e != nil {
		t.Errorf("shouldn't error: %s", e)
	}
	if nsec != "nsec180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsgyumg0" {
		t.Error("produced an unexpected nsec string")
	}
}

func TestDecodeNpub(t *testing.T) {
	prefix, pubkey, e := Decode("npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6")
	if e != nil {
		t.Errorf("shouldn't error: %s", e)
	}
	if prefix != "npub" {
		t.Error("returned invalid prefix")
	}
	if pubkey.(string) != "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d" {
		t.Error("returned wrong pubkey")
	}
}

func TestFailDecodeBadChecksumNpub(t *testing.T) {
	_, _, e := Decode("npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w4")
	if e == nil {
		t.Errorf("should have errored: %s", e)
	}
}

func TestDecodeNprofile(t *testing.T) {
	prefix, data, e := Decode("nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpp4mhxue69uhhytnc9e3k7mgpz4mhxue69uhkg6nzv9ejuumpv34kytnrdaksjlyr9p")
	if e != nil {
		t.Error("failed to decode nprofile")
	}
	if prefix != "nprofile" {
		t.Error("what")
	}
	pp, ok := data.(pointers.Profile)
	if !ok {
		t.Error("value returned of wrong type")
	}

	if pp.PublicKey != "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d" {
		t.Error("decoded invalid public key")
	}

	if len(pp.Relays) != 2 {
		t.Error("decoded wrong number of relays")
	}
	if pp.Relays[0] != "wss://r.x.com" || pp.Relays[1] != "wss://djbas.sadkb.com" {
		t.Error("decoded relay URLs wrongly")
	}
}

func TestDecodeOtherNprofile(t *testing.T) {
	prefix, data, e := Decode("nprofile1qqsw3dy8cpumpanud9dwd3xz254y0uu2m739x0x9jf4a9sgzjshaedcpr4mhxue69uhkummnw3ez6ur4vgh8wetvd3hhyer9wghxuet5qyw8wumn8ghj7mn0wd68yttjv4kxz7fww4h8get5dpezumt9qyvhwumn8ghj7un9d3shjetj9enxjct5dfskvtnrdakstl69hg")
	if e != nil {
		t.Error("failed to decode nprofile")
	}
	if prefix != "nprofile" {
		t.Error("what")
	}
	pp, ok := data.(pointers.Profile)
	if !ok {
		t.Error("value returned of wrong type")
	}

	if pp.PublicKey != "e8b487c079b0f67c695ae6c4c2552a47f38adfa2533cc5926bd2c102942fdcb7" {
		t.Error("decoded invalid public key")
	}

	if len(pp.Relays) != 3 {
		t.Error("decoded wrong number of relays")
	}
	if pp.Relays[0] != "wss://nostr-pub.wellorder.net" || pp.Relays[1] != "wss://nostr-relay.untethr.me" {
		t.Error("decoded relay URLs wrongly")
	}
}

func TestEncodeNprofile(t *testing.T) {
	nprofile, e := EncodeProfile("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", []string{
		"wss://r.x.com",
		"wss://djbas.sadkb.com",
	})
	if e != nil {
		t.Errorf("shouldn't error: %s", e)
	}
	if nprofile != "nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpp4mhxue69uhhytnc9e3k7mgpz4mhxue69uhkg6nzv9ejuumpv34kytnrdaksjlyr9p" {
		t.Error("produced an unexpected nprofile string")
	}
}

func TestEncodeDecodeNaddr(t *testing.T) {
	var naddr string
	var e error
	naddr, e = EncodeEntity(
		"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d",
		kind.Article,
		"banana",
		[]string{
			"wss://relay.nostr.example.mydomain.example.com",
			"wss://nostr.banana.com",
		})
	if e != nil {
		t.Errorf("shouldn't error: %s", e)
	}
	if naddr != "naddr1qqrxyctwv9hxzqfwwaehxw309aex2mrp0yhxummnw3ezuetcv9khqmr99ekhjer0d4skjm3wv4uxzmtsd3jjucm0d5q3vamnwvaz7tmwdaehgu3wvfskuctwvyhxxmmdqgsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8grqsqqqa28a3lkds" {
		t.Errorf("produced an unexpected naddr string: %s", naddr)
	}
	var prefix string
	var data any
	prefix, data, e = Decode(naddr)
	// log.D.S(prefix, data, e)
	if log.Fail(e) {
		t.Errorf("shouldn't error: %s", e)
	}
	if prefix != NentityHRP {
		t.Error("returned invalid prefix")
	}
	ep, ok := data.(pointers.Entity)
	if !ok {
		t.Fatalf("did not decode an entity type, got %v", reflect.TypeOf(data))
	}
	if ep.PublicKey != "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d" {
		t.Error("returned wrong pubkey")
	}
	if ep.Kind != kind.Article {
		t.Error("returned wrong kind")
	}
	if ep.Identifier != "banana" {
		t.Error("returned wrong identifier")
	}
	if ep.Relays[0] != "wss://relay.nostr.example.mydomain.example.com" || ep.Relays[1] != "wss://nostr.banana.com" {
		t.Error("returned wrong relays")
	}
}

func TestDecodeNaddrWithoutRelays(t *testing.T) {
	prefix, data, e := Decode("naddr1qq98yetxv4ex2mnrv4esygrl54h466tz4v0re4pyuavvxqptsejl0vxcmnhfl60z3rth2xkpjspsgqqqw4rsf34vl5")
	if e != nil {
		t.Errorf("shouldn't error: %s", e)
	}
	if prefix != "naddr" {
		t.Error("returned invalid prefix")
	}
	ep := data.(pointers.Entity)
	if ep.PublicKey != "7fa56f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751ac194" {
		t.Error("returned wrong pubkey")
	}
	if ep.Kind != kind.Article {
		t.Error("returned wrong kind")
	}
	if ep.Identifier != "references" {
		t.Error("returned wrong identifier")
	}
	if len(ep.Relays) != 0 {
		t.Error("relays should have been an empty array")
	}
}

func TestEncodeDecodeNEventTestEncodeDecodeNEvent(t *testing.T) {
	nevent, e := EncodeEvent(
		"45326f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751ac194",
		[]string{"wss://banana.com"},
		"7fa56f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751abb88",
	)
	if e != nil {
		t.Errorf("shouldn't error: %s", e)
	}

	prefix, res, e := Decode(nevent)
	if e != nil {
		t.Errorf("shouldn't error: %s", e)
	}

	if prefix != "nevent" {
		t.Errorf("should have 'nevent' prefix, not '%s'", prefix)
	}

	ep, ok := res.(pointers.Event)
	if !ok {
		t.Errorf("'%s' should be an nevent, not %v", nevent, res)
	}

	if ep.Author != "7fa56f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751abb88" {
		t.Error("wrong author")
	}

	if ep.ID != "45326f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751ac194" {
		t.Error("wrong id")
	}

	if len(ep.Relays) != 1 || ep.Relays[0] != "wss://banana.com" {
		t.Error("wrong relay")
	}
}
