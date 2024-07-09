// Package nostrbinary provides a simple interface for using Gob encoding on
// nostr events.
package nostrbinary

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/ec"
	"github.com/Hubmakerlabs/replicatr/pkg/ec/schnorr"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
	"github.com/minio/sha256-simd"
)

var log, chk = slog.New(os.Stderr)

// Event is the most compact and exact form of an event as encoded in native Go
// form. This will produce the most compact form of Gob encoded binary data.
type Event struct {
	ID        [sha256.Size]byte
	PubKey    [schnorr.PubKeyBytesLen]byte
	CreatedAt timestamp.T
	Kind      kind.T
	Tags      tags.T
	Content   string
	Sig       [schnorr.SignatureSize]byte
}

// FromEventT converts an event.T into an Event, including validating all fields
// are correctly formed for their type.
func FromEventT(evt *event.T) (evb *Event, err error) {
	if evt == nil {
		err = errors.New("nil event")
		return
	}
	evb = &Event{Kind: evt.Kind, Tags: evt.Tags,
		Content: evt.Content, CreatedAt: evt.CreatedAt}
	if len(evt.ID) != sha256.Size*2 {
		err = fmt.Errorf("incorrect event ID len, got %d expected %d",
			len(evt.ID), sha256.Size*2)
		return
	}
	copy(evb.ID[:], evt.ID.Bytes())
	if len(evt.ID) != schnorr.PubKeyBytesLen*2 {
		err = fmt.Errorf("incorrect event pubkey len, got %d expected %d",
			len(evt.PubKey), schnorr.PubKeyBytesLen*2)
		return
	}
	var pk []byte
	if pk, err = hex.Dec(evt.PubKey); chk.E(err) {
		return
	}
	var spk *ec.PublicKey
	if spk, err = schnorr.ParsePubKey(pk); chk.E(err) {
		return
	}
	copy(evb.PubKey[:], schnorr.SerializePubKey(spk))
	var sig []byte
	if len(evt.Sig) != schnorr.SignatureSize*2 {
		err = fmt.Errorf("incorrect event signature len, got %d expected %d",
			len(evt.Sig), schnorr.SignatureSize*2)
		return
	}
	if sig, err = hex.Dec(evt.Sig); chk.E(err) {
		return
	}
	var ss *schnorr.Signature
	if ss, err = schnorr.ParseSignature(sig); chk.E(err) {
		return
	}
	copy(evb.Sig[:], ss.Serialize())
	return
}

// ToEventT converts an Event into an event.T.
func (evb *Event) ToEventT() (evt *event.T) {
	evt = &event.T{
		ID:        eventid.T(hex.Enc(evb.ID[:])),
		PubKey:    hex.Enc(evb.PubKey[:]),
		CreatedAt: evb.CreatedAt,
		Kind:      evb.Kind,
		Tags:      evb.Tags,
		Content:   evb.Content,
		Sig:       hex.Enc(evb.Sig[:]),
	}
	return
}

func Unmarshal(data []byte) (evt *event.T, err error) {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	evb := &Event{}
	if err = dec.Decode(evb); chk.D(err) {
		return
	}
	evt = evb.ToEventT()
	return
}

func Marshal(evt *event.T) (b []byte, err error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	var evb *Event
	if evb, err = FromEventT(evt); chk.E(err) {
		return
	}
	if err = enc.Encode(evb); chk.D(err) {
		return
	}
	b = buf.Bytes()
	return
}
