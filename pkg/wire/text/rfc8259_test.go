package text

import (
	log2 "mleku.online/git/log"
	"testing"
)

const LastChar = '~'

func TestEscapeJSONStringAndWrap(t *testing.T) {
	log2.SetLogLevel(log2.Debug)
	escapeStringVersion := EscapeString([]byte{},
		GenerateStringWithAllASCII())
	escapeJSONStringAndWrapVersion :=
		EscapeJSONStringAndWrap(GenerateStringWithAllASCII())
	if len(escapeJSONStringAndWrapVersion) != len(escapeStringVersion) {
		t.Logf("escapeString version %d chars, "+
			"escapeJSONStringAndWrap version %d chars\n",
			len(escapeJSONStringAndWrapVersion), len(escapeStringVersion))
		t.Logf("escapeString\nlength: %d %s\n",
			len(escapeStringVersion), escapeStringVersion)
		t.Logf("escapJSONStringAndWrap\nlength: %d %s\n",
			len(escapeJSONStringAndWrapVersion),
			escapeJSONStringAndWrapVersion)
		t.FailNow()
	}
	for i := range escapeStringVersion {
		if escapeStringVersion[i] != escapeJSONStringAndWrapVersion[i] {
			t.Logf("escapeString version differs at index %d from "+
				"escapeJSONStringAndWrap version", i)
			t.Logf("escapeString\nlength: %d %s\n",
				len(escapeStringVersion), escapeStringVersion)
			t.Logf("escapJSONStringAndWrap\nlength: %d %s\n",
				len(escapeJSONStringAndWrapVersion),
				escapeJSONStringAndWrapVersion)
			t.Logf("got '%s' %d expected '%s' %d\n",
				string(escapeStringVersion[i]),
				escapeStringVersion[i],
				string(escapeJSONStringAndWrapVersion[i]),
				escapeJSONStringAndWrapVersion[i],
			)
			t.FailNow()
		}
	}
}

// GenerateStringWithAllASCII generates a string from code 0 up to
// 127:
func GenerateStringWithAllASCII() (str string) {
	const all = 1 << 7
	bytes := make([]byte, all)
	for i := range bytes {
		bytes[i] = byte(i)
	}
	return string(bytes)
}
