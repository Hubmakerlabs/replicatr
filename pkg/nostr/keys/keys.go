package keys

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/ec"
	"github.com/Hubmakerlabs/replicatr/pkg/ec/schnorr"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
)

func GeneratePrivateKey() string {
	params := btcec.S256().Params()
	one := new(big.Int).SetInt64(1)

	b := make([]byte, params.BitSize/8+8)
	if _, e := io.ReadFull(rand.Reader, b); e != nil {
		return ""
	}

	k := new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	k.Mod(k, n)
	k.Add(k, one)

	return fmt.Sprintf("%064x", k.Bytes())
}

func GetPublicKey(sk string) (string, error) {
	b, e := hex.Dec(sk)
	if e != nil {
		return "", e
	}

	_, pk := btcec.PrivKeyFromBytes(b)
	return hex.Enc(schnorr.SerializePubKey(pk)), nil
}

func IsValid32ByteHex(pk string) bool {
	if strings.ToLower(pk) != pk {
		return false
	}
	dec, _ := hex.Dec(pk)
	return len(dec) == 32
}
