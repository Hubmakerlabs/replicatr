package nip19

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"
	"github.com/Hubmakerlabs/replicatr/pkg/bech32"
)

const (
	NoteHRP     = "note"
	NsecHRP     = "nsec"
	NpubHRP     = "npub"
	NprofileHRP = "nprofile"
	NeventHRP   = "nevent"
	NentityHRP  = "naddr"
)

func DecodeToString(bech32String string) (prefix, value string, e error) {
	var s any
	if prefix, s, e = Decode(bech32String); fails(e) {
		return
	}
	var ok bool
	if value, ok = s.(string); ok {
		return
	}
	e = fmt.Errorf("value was not decoded to a string, found type %s",
		reflect.TypeOf(s))
	return
}

func Decode(bech32string string) (prefix string, value any, e error) {
	var bits5 []byte
	if prefix, bits5, e = bech32.DecodeNoLimit(bech32string); fails(e) {
		return
	}
	var data []byte
	if data, e = bech32.ConvertBits(bits5, 5, 8, false); fails(e) {
		return prefix, nil, fmt.Errorf("failed translating data into 8 bits: %s",
			e.Error())
	}
	switch prefix {
	case NpubHRP, NsecHRP, NoteHRP:
		if len(data) < 32 {
			return prefix, nil, fmt.Errorf("data is less than 32 bytes (%d)",
				len(data))
		}
		return prefix, hex.EncodeToString(data[0:32]), nil
	case NprofileHRP:
		var result pointers.Profile
		curr := 0
		for {
			t, v := readTLVEntry(data[curr:])
			if v == nil {
				// end here
				if result.PublicKey == "" {
					return prefix, result, fmt.Errorf("no pubkey found for nprofile")
				}
				return prefix, result, nil
			}
			switch t {
			case TLVDefault:
				if len(v) < 32 {
					return prefix, nil, fmt.Errorf("pubkey is less than 32 bytes (%d)",
						len(v))
				}
				result.PublicKey = hex.EncodeToString(v)
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			default:
				// ignore
			}
			curr = curr + 2 + len(v)
		}
	case NeventHRP:
		var result pointers.Event
		curr := 0
		for {
			t, v := readTLVEntry(data[curr:])
			if v == nil {
				// end here
				if result.ID == "" {
					return prefix, result, fmt.Errorf("no id found for nevent")
				}
				return prefix, result, nil
			}
			switch t {
			case TLVDefault:
				if len(v) < 32 {
					return prefix, nil, fmt.Errorf("id is less than 32 bytes (%d)",
						len(v))
				}
				result.ID = eventid.EventID(hex.EncodeToString(v))
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			case TLVAuthor:
				if len(v) < 32 {
					return prefix, nil, fmt.Errorf("author is less than 32 bytes (%d)",
						len(v))
				}
				result.Author = hex.EncodeToString(v)
			case TLVKind:
				result.Kind = kind.T(binary.BigEndian.Uint32(v))
			default:
				// ignore
			}
			curr = curr + 2 + len(v)
		}
	case NentityHRP:
		var result pointers.Entity
		curr := 0
		for {
			t, v := readTLVEntry(data[curr:])
			if v == nil {
				// log.D.S(t, v)
				// end here
				if result.Kind == 0 || result.Identifier == "" || result.PublicKey == "" {
					return prefix, result, fmt.Errorf("incomplete naddr")
				}

				return prefix, result, nil
			}
			switch t {
			case TLVDefault:
				result.Identifier = string(v)
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			case TLVAuthor:
				if len(v) < 32 {
					return prefix, nil, fmt.Errorf("author is less than 32 bytes (%d)", len(v))
				}
				result.PublicKey = hex.EncodeToString(v)
			case TLVKind:
				result.Kind = kind.T(binary.BigEndian.Uint32(v))
			default:
				log.D.Ln("got a bogus TLV type code", t)
				// ignore
			}
			curr = curr + 2 + len(v)
		}
	}
	return prefix, data, fmt.Errorf("unknown tag %s", prefix)
}

func EncodePrivateKey(privateKeyHex string) (string, error) {
	b, e := hex.DecodeString(privateKeyHex)
	if e != nil {
		return "", fmt.Errorf("failed to decode private key hex: %w", e)
	}
	bits5, e := bech32.ConvertBits(b, 8, 5, true)
	if e != nil {
		return "", e
	}
	return bech32.Encode(NsecHRP, bits5)
}

func EncodePublicKey(publicKeyHex string) (s string, e error) {
	var b []byte
	if b, e = hex.DecodeString(publicKeyHex); fails(e) {
		e = fmt.Errorf("failed to decode public key hex: %w", e)
		return
	}
	var bits5 []byte
	bits5, e = bech32.ConvertBits(b, 8, 5, true)
	if e != nil {
		return "", e
	}
	return bech32.Encode(NpubHRP, bits5)
}

func EncodeNote(eventIDHex string) (s string, e error) {
	var b []byte
	if b, e = hex.DecodeString(eventIDHex); fails(e) {
		e = fmt.Errorf("failed to decode event id hex: %w", e)
		return
	}
	var bits5 []byte
	if bits5, e = bech32.ConvertBits(b, 8, 5, true); fails(e) {
		return
	}
	return bech32.Encode(NoteHRP, bits5)
}

func EncodeProfile(publicKeyHex string, relays []string) (s string, e error) {
	buf := &bytes.Buffer{}
	var pb []byte
	if pb, e = hex.DecodeString(publicKeyHex); fails(e) {
		e = fmt.Errorf("invalid pubkey '%s': %w", publicKeyHex, e)
		return
	}
	writeTLVEntry(buf, TLVDefault, pb)
	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}
	var bits5 []byte
	if bits5, e = bech32.ConvertBits(buf.Bytes(), 8, 5, true); fails(e) {
		e = fmt.Errorf("failed to convert bits: %w", e)
		return
	}
	return bech32.Encode(NprofileHRP, bits5)
}

func EncodeEvent(eventIDHex string, relays []string,
	author string) (s string, e error) {

	buf := &bytes.Buffer{}
	var id []byte
	id, e = hex.DecodeString(eventIDHex)
	if e != nil || len(id) != 32 {
		return "", fmt.Errorf("invalid id '%s': %w", eventIDHex, e)
	}
	writeTLVEntry(buf, TLVDefault, id)
	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}
	if pubkey, _ := hex.DecodeString(author); len(pubkey) == 32 {
		writeTLVEntry(buf, TLVAuthor, pubkey)
	}
	var bits5 []byte
	if bits5, e = bech32.ConvertBits(buf.Bytes(), 8, 5, true); fails(e) {
		e = fmt.Errorf("failed to convert bits: %w", e)
		return
	}

	return bech32.Encode(NeventHRP, bits5)
}

func EncodeEntity(publicKey string, kind kind.T, identifier string,
	relays []string) (s string, e error) {

	buf := &bytes.Buffer{}
	writeTLVEntry(buf, TLVDefault, []byte(identifier))
	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}
	var pb []byte
	pb, e = hex.DecodeString(publicKey)
	if e != nil {
		return "", fmt.Errorf("invalid pubkey '%s': %w", pb, e)
	}
	writeTLVEntry(buf, TLVAuthor, pb)
	kindBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(kindBytes, uint32(kind))
	writeTLVEntry(buf, TLVKind, kindBytes)
	var bits5 []byte
	if bits5, e = bech32.ConvertBits(buf.Bytes(), 8, 5, true); fails(e) {
		return "", fmt.Errorf("failed to convert bits: %w", e)
	}
	return bech32.Encode(NentityHRP, bits5)
}
