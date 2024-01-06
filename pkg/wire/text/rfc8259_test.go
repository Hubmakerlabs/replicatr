package text

import (
	"testing"

	"github.com/minio/sha256-simd"
	"lukechampine.com/frand"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
)

func GenRandString(l int, src *frand.RNG) (str string) {
	return string(src.Bytes(l))
}

var seed = sha256.Sum256([]byte(`
The tao that can be told
is not the eternal Tao
The name that can be named
is not the eternal Name.

The unnamable is the eternally real.
Naming is the origin
of all particular things.

Free from desire, you realize the mystery.
Caught in desire, you see only the manifestations.

Yet mystery and manifestations
arise from the same source.
This source is called darkness.

Darkness within darkness.
The gateway to all understanding.
`))

var src = frand.NewCustom(seed[:], 32, 12)

func TestRandomEscapeJSONStringAndWrap(t *testing.T) {
	// this is a kind of fuzz test, does a massive number of iterations of
	// random content that ensures the escaping is correct without creating a
	// fixed set of test vectors.

	log2.SetLogLevel(log2.Debug)
	for i := 0; i < 1000; i++ {
		l := src.Intn(1<<8) + 32
		s1 := GenRandString(l, src)
		s2 := make([]byte, l)
		orig := make([]byte, l)
		copy(s2, s1)
		copy(orig, s1)

		// first we are checking our implementation comports to the one from go-nostr.
		escapeStringVersion := EscapeString([]byte{}, s1)
		escapeJSONStringAndWrapVersion :=
			EscapeJSONStringAndWrap(string(s2))
		if len(escapeJSONStringAndWrapVersion) != len(escapeStringVersion) {
			t.Logf("escapeString version %d chars, "+
				"escapeJSONStringAndWrap version %d chars\n",
				len(escapeJSONStringAndWrapVersion), len(escapeStringVersion))
			t.Logf("escapeString\nlength: %d\n%v\n",
				len(escapeStringVersion), escapeStringVersion)
			t.Logf("escapJSONStringAndWrap\nlength: %d\n%v\n",
				len(escapeJSONStringAndWrapVersion),
				escapeJSONStringAndWrapVersion)
			t.Logf("escapeString\nlength: %d\n%s\n%v\n",
				len(escapeStringVersion), escapeStringVersion,
				escapeStringVersion)
			t.Logf("escapJSONStringAndWrap\nlength: %d\n%s\n%v\n",
				len(escapeJSONStringAndWrapVersion),
				escapeJSONStringAndWrapVersion,
				escapeJSONStringAndWrapVersion)
			t.FailNow()
		}
		for i := range escapeStringVersion {
			if i > len(escapeJSONStringAndWrapVersion) {
				t.Fatal("escapeString version is shorter")
			}
			if escapeStringVersion[i] != escapeJSONStringAndWrapVersion[i] {
				t.Logf("escapeString version differs at index %d from "+
					"escapeJSONStringAndWrap version\n%s\n%s\n%v\n%v", i,
					escapeStringVersion[i-4:],
					escapeJSONStringAndWrapVersion[i-4:],
					escapeStringVersion[i-4:],
					escapeJSONStringAndWrapVersion[i-4:])
				t.Logf("escapeString\nlength: %d %s\n",
					len(escapeStringVersion), escapeStringVersion)
				t.Logf("escapJSONStringAndWrap\nlength: %d %s\n",
					len(escapeJSONStringAndWrapVersion),
					escapeJSONStringAndWrapVersion)
				t.Logf("got '%s' %d expected '%s' %d\n",
					string(escapeJSONStringAndWrapVersion[i]),
					escapeJSONStringAndWrapVersion[i],
					string(escapeStringVersion[i]),
					escapeStringVersion[i],
				)
				t.FailNow()
			}
		}

		// next, unescape the output and see if it matches the original
		unescaped := UnescapeByteString(Unwrap(escapeJSONStringAndWrapVersion))
		// t.Logf("unescaped: \n%s\noriginal:  \n%s", unescaped, orig)
		if string(unescaped) != string(orig) {
			t.Fatalf("\ngot      %d %v\nexpected %d %v\n",
				len(unescaped),
				unescaped,
				len(orig),
				orig,
			)
		}
	}
}
