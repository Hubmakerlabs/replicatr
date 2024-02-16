package nip44

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/hkdf"
	"mleku.online/git/ec/secp256k1"
	"mleku.online/git/slog"
)

var log, chk = slog.New(os.Stderr)

const (
	Reserved int = iota
	Deprecated1
	Standard1
)

var (
	MinPlaintextSize = 0x0001 // 1b msg => padded to 32b
	MaxPlaintextSize = 0xffff // 65535 (64kb-1) => padded to 64kb
)

type EncryptOptions struct {
	Salt    []byte
	Version int
}

func Encrypt(conversationKey []byte, plaintext string, options *EncryptOptions) (string, error) {
	var (
		version    int = Standard1
		salt       []byte
		enc        []byte
		nonce      []byte
		auth       []byte
		padded     []byte
		ciphertext []byte
		hmac_      []byte
		concat     []byte
		err        error
	)
	if options.Version != 0 {
		version = options.Version
	}
	if options.Salt != nil {
		salt = options.Salt
	} else {
		if salt, err = randomBytes(32); chk.E(err) {
			return "", err
		}
	}
	if version != 2 {
		return "", errors.New(fmt.Sprintf("unknown version %d", version))
	}
	if len(salt) != 32 {
		return "", errors.New("salt must be 32 bytes")
	}
	if enc, nonce, auth, err = messageKeys(conversationKey, salt); chk.E(err) {
		return "", err
	}
	if padded, err = pad(plaintext); chk.E(err) {
		return "", err
	}
	if ciphertext, err = chacha20_(enc, nonce, []byte(padded)); chk.E(err) {
		return "", err
	}
	if hmac_, err = sha256Hmac(auth, ciphertext, salt); chk.E(err) {
		return "", err
	}
	concat = append(concat, []byte{byte(version)}...)
	concat = append(concat, salt...)
	concat = append(concat, ciphertext...)
	concat = append(concat, hmac_...)
	return base64.StdEncoding.EncodeToString(concat), nil
}

func Decrypt(conversationKey []byte, ciphertext string) (string, error) {
	var (
		version     int = 2
		dcd         []byte
		cLen        int
		dLen        int
		salt        []byte
		ciphertext_ []byte
		hmac        []byte
		hmac_       []byte
		enc         []byte
		nonce       []byte
		auth        []byte
		padded      []byte
		unpaddedLen uint16
		unpadded    []byte
		err         error
	)
	cLen = len(ciphertext)
	if cLen < 132 || cLen > 87472 {
		err = errors.New(fmt.Sprintf("invalid payload length: %d", cLen))
		log.E.Ln(err)
		return "", err
	}
	if ciphertext[0:1] == "#" {
		err = errors.New("unknown version")
		log.E.Ln(err)
		return "", err
	}
	if dcd, err = base64.StdEncoding.DecodeString(ciphertext); chk.E(err) {
		err = errors.New("invalid base64")
		log.E.Ln(err)
		return "", err
	}
	log.D.Ln("decoded", dcd)
	if version = int(dcd[0]); version != 2 {
		err = errors.New(fmt.Sprintf("unknown version %d", version))
		log.E.Ln(err)
		return "", err
	}
	dLen = len(dcd)
	if dLen < 99 || dLen > 65603 {
		err = errors.New(fmt.Sprintf("invalid data length: %d", dLen))
		log.E.Ln(err)
		return "", err
	}
	salt, ciphertext_, hmac_ = dcd[1:33], dcd[33:dLen-32], dcd[dLen-32:]
	log.D.Ln(salt, ciphertext_, hmac_)
	if enc, nonce, auth, err = messageKeys(conversationKey, salt); chk.E(err) {
		return "", err
	}
	if hmac, err = sha256Hmac(auth, ciphertext_, salt); chk.E(err) {
		return "", err
	}
	if !bytes.Equal(hmac_, hmac) {
		return "", errors.New("invalid hmac")
	}
	if padded, err = chacha20_(enc, nonce, ciphertext_); chk.E(err) {
		return "", err
	}
	unpaddedLen = binary.BigEndian.Uint16(padded[0:2])
	if unpaddedLen < uint16(MinPlaintextSize) || unpaddedLen > uint16(MaxPlaintextSize) || len(padded) != 2+calcPadding(int(unpaddedLen)) {
		err = errors.New("invalid padding")
		return "", err
	}
	unpadded = padded[2 : unpaddedLen+2]
	if len(unpadded) == 0 || len(unpadded) != int(unpaddedLen) {
		err = errors.New("invalid padding")
		log.D.Ln(err)
		return "", err
	}
	return string(unpadded), nil
}

func GenerateConversationKey(sendPrivkey *secp256k1.PrivateKey, recvPubkey *secp256k1.PublicKey) []byte {
	// TODO: Make sure keys are not invalid or weak since the secp256k1 package does not.
	// See documentation of secp256k1.PrivKeyFromBytes:
	// ================================================================================
	// | WARNING: This means passing a slice with more than 32 bytes is truncated and |
	// | that truncated value is reduced modulo N.  Further, 0 is not a valid private |
	// | key.  It is up to the caller to provide a value in the appropriate range of  |
	// | [1, N-1].  Failure to do so will either result in an invalid private key or  |
	// | potentially weak private keys that have bias that could be exploited.        |
	// ================================================================================
	// -- https://pkg.go.dev/github.com/decred/dcrd/dcrec/secp256k1/v4#PrivKeyFromBytes
	shared := secp256k1.GenerateSharedSecret(sendPrivkey, recvPubkey)
	return hkdf.Extract(sha256.New, shared, []byte("nip44-v2"))
}

func chacha20_(key []byte, nonce []byte, message []byte) ([]byte, error) {
	var (
		cipher *chacha20.Cipher
		dst    = make([]byte, len(message))
		err    error
	)
	if cipher, err = chacha20.NewUnauthenticatedCipher(key, nonce); chk.E(err) {
		return nil, err
	}
	cipher.XORKeyStream(dst, message)
	return dst, nil
}

func randomBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); chk.E(err) {
		return nil, err
	}
	return buf, nil
}

func sha256Hmac(key []byte, ciphertext []byte, aad []byte) ([]byte, error) {
	if len(aad) != 32 {
		return nil, errors.New("aad data must be 32 bytes")
	}
	h := hmac.New(sha256.New, key)
	h.Write(aad)
	h.Write(ciphertext)
	return h.Sum(nil), nil
}

func messageKeys(conversationKey []byte, salt []byte) ([]byte, []byte, []byte, error) {
	var (
		r     io.Reader
		enc   []byte = make([]byte, 32)
		nonce []byte = make([]byte, 12)
		auth  []byte = make([]byte, 32)
		err   error
	)
	if len(conversationKey) != 32 {
		return nil, nil, nil, errors.New("conversation key must be 32 bytes")
	}
	if len(salt) != 32 {
		return nil, nil, nil, errors.New("salt must be 32 bytes")
	}
	r = hkdf.Expand(sha256.New, conversationKey, salt)
	if _, err = io.ReadFull(r, enc); chk.E(err) {
		return nil, nil, nil, err
	}
	if _, err = io.ReadFull(r, nonce); chk.E(err) {
		return nil, nil, nil, err
	}
	if _, err = io.ReadFull(r, auth); chk.E(err) {
		return nil, nil, nil, err
	}
	return enc, nonce, auth, nil
}

func pad(s string) ([]byte, error) {
	var (
		sb      []byte
		sbLen   int
		padding int
		result  []byte
	)
	sb = []byte(s)
	sbLen = len(sb)
	if sbLen < 1 || sbLen > MaxPlaintextSize {
		return nil, errors.New("plaintext should be between 1b and 64kB")
	}
	padding = calcPadding(sbLen)
	result = make([]byte, 2)
	binary.BigEndian.PutUint16(result, uint16(sbLen))
	result = append(result, sb...)
	result = append(result, make([]byte, padding-sbLen)...)
	return result, nil
}

func calcPadding(sLen int) int {
	var (
		nextPower int
		chunk     int
	)
	if sLen <= 32 {
		return 32
	}
	nextPower = 1 << int(math.Floor(math.Log2(float64(sLen-1)))+1)
	chunk = int(math.Max(32, float64(nextPower/8)))
	return chunk * int(math.Floor(float64((sLen-1)/chunk))+1)
}
