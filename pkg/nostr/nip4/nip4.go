package nip4

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
	secp "mleku.online/git/ec/secp"
	log2 "mleku.online/git/log"
	"strings"
)

var (
	log                    = log2.GetLogger()
	fails                  = log.D.Chk
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

// ComputeSharedSecret computes an Elliptic Curve Diffie Hellman shared secret
// out of one public key and another secret key.
//
// The public key and secret key for this can be either hex or bech32 formatted,
// since this is easily determined by reading the first 4 bytes of the string
func ComputeSharedSecret(pub string, sec string) (secret []byte, e error) {
	if len(pub) < nip19.MinKeyStringLen {
		e = fmt.Errorf("public key is too short, must be at least %d, "+
			"'%s' is only %d chars", nip19.MinKeyStringLen, pub, len(pub))
		return
	}
	if len(sec) < nip19.MinKeyStringLen {
		e = fmt.Errorf("public key is too short, must be at least %d, "+
			"'%s' is only %d chars", nip19.MinKeyStringLen, pub, len(pub))
		return
	}
	var s *secp.SecretKey
	var p *secp.PublicKey
	// if the first 4 chars are a Bech32 HRP try to decode as Bech32
	if pub[:nip19.Bech32HRPLen] == nip19.PubHRP {
		if p, e = nip19.NpubToPublicKey(pub); fails(e) {
			return
		}
	} else {
		if p, e = nip19.HexToPublicKey(pub); fails(e) {
			return
		}
	}
	// if the first 4 chars are a Bech32 HRP try to decode as Bech32
	if sec[:nip19.Bech32HRPLen] == nip19.SecHRP {
		if s, e = nip19.NsecToSecretKey(sec); fails(e) {
			return
		}
	} else {
		if s, e = nip19.HexToSecretKey(sec); fails(e) {
			return
		}
	}
	return secp.GenerateSharedSecret(s, p), e
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
	// can probably use a less expensive lib since IV has to only be unique; not
	// perfectly random; math/rand? ed: https://github.com/lukechampine/frand
	// but this is not high volume throughput and only one good IV is needed per
	// 4gb of data at most.
	if _, e := rand.Read(iv); e != nil {
		return "", fmt.Errorf("error creating initization vector: %w", e)
	}
	// automatically picks aes-256 based on key length (32 bytes)
	block, e := aes.NewCipher(key)
	if e != nil {
		return "", fmt.Errorf("error creating block cipher: %w", e)
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	plaintext := []byte(message)
	// add padding
	base := len(plaintext)
	// this will be a number between 1 and 16 (including), never 0
	padding := block.BlockSize() - base%block.BlockSize()
	// encode the padding in all the padding bytes themselves
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	paddedMsgBytes := append(plaintext, padtext...)
	ciphertext := make([]byte, len(paddedMsgBytes))
	mode.CryptBlocks(ciphertext, paddedMsgBytes)
	return base64.StdEncoding.EncodeToString(ciphertext) + "?iv=" +
		base64.StdEncoding.EncodeToString(iv), nil
}

// Decrypt decrypts a content string using the shared secret key.
// The inverse operation to message -> Encrypt(message, key).
func Decrypt(content string, key []byte) (string, error) {
	parts := strings.Split(content, "?iv=")
	if len(parts) < 2 {
		return "", fmt.Errorf(
			"error parsing encrypted message: no initialization vector")
	}
	ciphertext, e := base64.StdEncoding.DecodeString(parts[0])
	if e != nil {
		return "", fmt.Errorf(
			"error decoding ciphertext from base64: %w", e)
	}
	iv, e := base64.StdEncoding.DecodeString(parts[1])
	if e != nil {
		return "", fmt.Errorf("error decoding iv from base64: %w", e)
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return "", fmt.Errorf("error creating block cipher: %w", e)
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)
	// remove padding
	var (
		message      = string(plaintext)
		plaintextLen = len(plaintext)
	)
	if plaintextLen > 0 {
		// the padding amount is encoded in the padding bytes themselves
		padding := int(plaintext[plaintextLen-1])
		if padding > plaintextLen {
			return "", fmt.Errorf("invalid padding amount: %d", padding)
		}
		message = string(plaintext[0 : plaintextLen-padding])
	}
	return message, nil
}
