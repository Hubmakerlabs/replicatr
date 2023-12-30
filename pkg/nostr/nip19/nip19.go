package nip19

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"
	"mleku.online/git/bech32"
)

func Decode(bech32string string) (prefix string, value any, e error) {
	prefix, bits5, e := bech32.DecodeNoLimit(bech32string)
	if e != nil {
		return "", nil, e
	}

	data, e := bech32.ConvertBits(bits5, 5, 8, false)
	if e != nil {
		return prefix, nil, fmt.Errorf("failed translating data into 8 bits: %s", e.Error())
	}

	switch prefix {
	case "npub", "nsec", "note":
		if len(data) < 32 {
			return prefix, nil, fmt.Errorf("data is less than 32 bytes (%d)", len(data))
		}

		return prefix, hex.EncodeToString(data[0:32]), nil
	case "nprofile":
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
					return prefix, nil, fmt.Errorf("pubkey is less than 32 bytes (%d)", len(v))
				}
				result.PublicKey = hex.EncodeToString(v)
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			default:
				// ignore
			}

			curr = curr + 2 + len(v)
		}
	case "nevent":
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
					return prefix, nil, fmt.Errorf("id is less than 32 bytes (%d)", len(v))
				}
				result.ID = nip1.EventID(hex.EncodeToString(v))
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			case TLVAuthor:
				if len(v) < 32 {
					return prefix, nil, fmt.Errorf("author is less than 32 bytes (%d)", len(v))
				}
				result.Author = hex.EncodeToString(v)
			case TLVKind:
				result.Kind = kind.T(binary.BigEndian.Uint32(v))
			default:
				// ignore
			}

			curr = curr + 2 + len(v)
		}
	case "naddr":
		var result pointers.Entity
		curr := 0
		for {
			t, v := readTLVEntry(data[curr:])
			if v == nil {
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

	return bech32.Encode("nsec", bits5)
}

func EncodePublicKey(publicKeyHex string) (string, error) {
	b, e := hex.DecodeString(publicKeyHex)
	if e != nil {
		return "", fmt.Errorf("failed to decode public key hex: %w", e)
	}

	bits5, e := bech32.ConvertBits(b, 8, 5, true)
	if e != nil {
		return "", e
	}

	return bech32.Encode("npub", bits5)
}

func EncodeNote(eventIDHex string) (string, error) {
	b, e := hex.DecodeString(eventIDHex)
	if e != nil {
		return "", fmt.Errorf("failed to decode event id hex: %w", e)
	}

	bits5, e := bech32.ConvertBits(b, 8, 5, true)
	if e != nil {
		return "", e
	}

	return bech32.Encode("note", bits5)
}

func EncodeProfile(publicKeyHex string, relays []string) (string, error) {
	buf := &bytes.Buffer{}
	pubkey, e := hex.DecodeString(publicKeyHex)
	if e != nil {
		return "", fmt.Errorf("invalid pubkey '%s': %w", publicKeyHex, e)
	}
	writeTLVEntry(buf, TLVDefault, pubkey)

	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}

	bits5, e := bech32.ConvertBits(buf.Bytes(), 8, 5, true)
	if e != nil {
		return "", fmt.Errorf("failed to convert bits: %w", e)
	}

	return bech32.Encode("nprofile", bits5)
}

func EncodeEvent(eventIDHex string, relays []string, author string) (string, error) {
	buf := &bytes.Buffer{}
	id, e := hex.DecodeString(eventIDHex)
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

	bits5, e := bech32.ConvertBits(buf.Bytes(), 8, 5, true)
	if e != nil {
		return "", fmt.Errorf("failed to convert bits: %w", e)
	}

	return bech32.Encode("nevent", bits5)
}

func EncodeEntity(publicKey string, kind kind.T, identifier string, relays []string) (string, error) {
	buf := &bytes.Buffer{}

	writeTLVEntry(buf, TLVDefault, []byte(identifier))

	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}

	pubkey, e := hex.DecodeString(publicKey)
	if e != nil {
		return "", fmt.Errorf("invalid pubkey '%s': %w", pubkey, e)
	}
	writeTLVEntry(buf, TLVAuthor, pubkey)

	kindBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(kindBytes, uint32(kind))
	writeTLVEntry(buf, TLVKind, kindBytes)

	bits5, e := bech32.ConvertBits(buf.Bytes(), 8, 5, true)
	if e != nil {
		return "", fmt.Errorf("failed to convert bits: %w", e)
	}

	return bech32.Encode("naddr", bits5)
}
