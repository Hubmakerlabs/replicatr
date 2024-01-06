package nip04

import (
	"strings"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
)

func TestEncryptionAndDecryption(t *testing.T) {
	sharedSecret := make([]byte, 32)
	message := "hello hello"

	ciphertext, e := Encrypt(message, sharedSecret)
	if e != nil {
		t.Errorf("failed to encrypt: %s", e.Error())
	}

	plaintext, e := Decrypt(ciphertext, sharedSecret)
	if e != nil {
		t.Errorf("failed to decrypt: %s", e.Error())
	}

	if message != plaintext {
		t.Errorf("original '%s' and decrypted '%s' messages differ", message, plaintext)
	}
}

func TestEncryptionAndDecryptionWithMultipleLengths(t *testing.T) {
	sharedSecret := make([]byte, 32)

	for i := 0; i < 150; i++ {
		message := strings.Repeat("a", i)

		ciphertext, e := Encrypt(message, sharedSecret)
		if e != nil {
			t.Errorf("failed to encrypt: %s", e.Error())
		}

		plaintext, e := Decrypt(ciphertext, sharedSecret)
		if e != nil {
			t.Errorf("failed to decrypt: %s", e.Error())
		}

		if message != plaintext {
			t.Errorf("original '%s' and decrypted '%s' messages differ", message, plaintext)
		}
	}
}

func TestNostrToolsCompatibility(t *testing.T) {
	sk1 := "92996316beebf94171065a714cbf164d1f56d7ad9b35b329d9fc97535bf25352"
	sk2 := "591c0c249adfb9346f8d37dfeed65725e2eea1d7a6e99fa503342f367138de84"
	pk2, _ := keys.GetPublicKey(sk2)
	shared, _ := ComputeSharedSecret(pk2, sk1)
	ciphertext := "A+fRnU4aXS4kbTLfowqAww==?iv=QFYUrl5or/n/qamY79ze0A=="
	plaintext, _ := Decrypt(ciphertext, shared)
	if plaintext != "hello" {
		t.Fatal("invalid decryption of nostr-tools payload")
	}
}
