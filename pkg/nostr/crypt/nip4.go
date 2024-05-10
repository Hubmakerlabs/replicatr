package crypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	b32 "github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"lukechampine.com/frand"
	secp "mleku.dev/git/ec/secp256k1"
	"mleku.dev/git/slog"
)

var (
	log, chk = slog.New(os.Stderr)
	errf     = fmt.Errorf
)

// ComputeSharedSecret computes an Elliptic Curve Diffie Hellman shared secret
// out of one public key and another secret key.
//
// The public key and secret key for this can be either hex or bech32 formatted,
// since this is easily determined by reading the first 4 bytes of the string
func ComputeSharedSecret(sec, pub string) (secret []byte, err error) {
	if len(pub) < b32.MinKeyStringLen {
		err = log.E.Err("public key is too short, must be at least %d, "+
			"'%s' is only %d chars",
			b32.MinKeyStringLen, pub, len(pub))
		return
	}
	if len(sec) < b32.MinKeyStringLen {
		err = log.E.Err("public key is too short, must be at least %d, "+
			"'%s' is only %d chars",
			b32.MinKeyStringLen, pub, len(pub))
		return
	}
	var s *secp.SecretKey
	var p *secp.PublicKey
	// if the first 4 chars are a Bech32 HRP try to decode as Bech32

	if strings.HasPrefix(pub, b32.PubHRP) {
		if p, err = b32.NpubToPublicKey(pub); chk.D(err) {
			return
		}
	} else {
		if p, err = b32.HexToPublicKey(pub); chk.D(err) {
			return
		}
	}
	// if the first 4 chars are a Bech32 HRP try to decode as Bech32
	if strings.HasPrefix(sec, b32.SecHRP) {
		if s, err = b32.NsecToSecretKey(sec); chk.D(err) {
			return
		}
	} else {
		if s, err = b32.HexToSecretKey(sec); chk.D(err) {
			return
		}
	}
	return secp.GenerateSharedSecret(s, p), err
}

func GenerateSharedSecret(s *secp.SecretKey, p *secp.PublicKey) []byte {
	return secp.GenerateSharedSecret(s, p)
}

// Encrypt encrypts message with key using aes-256-cbc. key should be the shared
// secret generated by ComputeSharedSecret.
//
// Returns: base64(encrypted_bytes) + "?iv=" + base64(initialization_vector).
func Encrypt(message string, key []byte) (string, error) {
	// block size is 16 bytes
	iv := make([]byte, 16)
	if _, err := frand.Read(iv); err != nil {
		return "", errf("error creating initization vector: %w", err)
	}
	// automatically picks aes-256 based on key length (32 bytes)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errf("error creating block cipher: %w", err)
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	plaintext := []byte(message)
	// add padding
	base := len(plaintext)
	// this will be a number between 1 and 16 (inclusive), never 0
	bs := block.BlockSize()
	padding := bs - base%bs
	// encode the padding in all the padding bytes themselves
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	paddedMsgBytes := append(plaintext, padText...)
	ciphertext := make([]byte, len(paddedMsgBytes))
	mode.CryptBlocks(ciphertext, paddedMsgBytes)
	return base64.StdEncoding.EncodeToString(ciphertext) + "?iv=" +
		base64.StdEncoding.EncodeToString(iv), nil
}

// Decrypt decrypts a content string using the shared secret key.
// The inverse operation to message -> Encrypt(message, key).
func Decrypt(content string, key []byte) ([]byte, error) {
	parts := strings.Split(content, "?iv=")
	if len(parts) < 2 {
		return nil, errf(
			"error parsing encrypted message: no initialization vector")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errf(
			"error decoding ciphertext from base64: %w", err)
	}
	var iv []byte
	iv, err = base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errf("error decoding iv from base64: %w", err)
	}
	var block cipher.Block
	block, err = aes.NewCipher(key)
	if err != nil {
		return nil, errf("error creating block cipher: %w", err)
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)
	// remove padding
	var (
		message      = plaintext
		plaintextLen = len(plaintext)
	)
	if plaintextLen > 0 {
		// the padding amount is encoded in the padding bytes themselves
		padding := int(plaintext[plaintextLen-1])
		if padding > plaintextLen {
			return nil, errf("invalid padding amount: %d", padding)
		}
		message = plaintext[0 : plaintextLen-padding]
	}
	return message, nil
}
