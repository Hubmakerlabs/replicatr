package jsontext

import "testing"

const LastChar = '~'

func TestEscapeJSONStringAndWrap(t *testing.T) {
	escapeStringVersion := EscapeString([]byte{},
		GenerateStringWithAllASCII())
	escapeJSONStringAndWrapVersion :=
		EscapeJSONStringAndWrap(GenerateStringWithAllASCII())
	if len(escapeJSONStringAndWrapVersion) != len(escapeStringVersion) {
		t.Logf("escapeString version %d chars, "+
			"escapeJSONStringAndWrap version %d chars\n",
			len(escapeJSONStringAndWrapVersion), len(escapeStringVersion))
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
	const all = 1 << 8
	bytes := make([]byte, all)
	for i := range bytes {
		bytes[i] = byte(i)
	}
	return string(bytes)
}
