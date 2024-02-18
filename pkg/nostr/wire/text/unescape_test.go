package text

import (
	"fmt"
	"testing"

	"mleku.dev/git/slog"
)

func TestUnescapeByteString(t *testing.T) {
	slog.SetLogLevel(slog.Debug)
	b := make([]byte, 256)
	for i := range b {
		b[i] = byte(i)
	}
	escaped := make([]byte, 256*6)
	for i := range b {
		copy(escaped[i*6:i*6+6], fmt.Sprintf("\\u00%02x", i))
	}
	// log.D.F("Original               %3d\n\n'%s' '%s'",
	// 	len(b), string(b[:128]), string(b[128:]))
	// log.D.Ln(string(escaped))
	unescaped := UnescapeByteString(escaped)
	// log.D.Ln(len(b), string(b))
	// log.D.Ln(len(unescaped), string(unescaped))
	if string(unescaped) != string(b) {
		t.Fatalf("mismatched from original after unescaping:\n%v\n%v",
			b, unescaped)
	}
	jsonEscaped := EscapeJSONStringAndWrap(string(b))
	jsonEscaped = jsonEscaped[1 : len(jsonEscaped)-1]
	// log.D.F("JSON Escaped           %3d '%s'",
	// 	len(jsonEscaped), string(jsonEscaped))
	jsonUnescaped := UnescapeByteString(jsonEscaped)
	// log.D.F("Unescaped Escaped JSON %3d\n\n'%s'",
	// 	len(jsonUnescaped), string(jsonUnescaped))
	if len(b) != len(jsonUnescaped) {
		t.Fatalf("mismatch of original and unescaped strings: expected %d, got %d",
			len(b), len(jsonUnescaped))
	}
	var failed bool
	for i := range b {
		if b[i] != jsonUnescaped[i] {
			t.Logf("mismatch of charater in output, index %d got %d expected %d",
				i, b[i], jsonUnescaped[i])
			failed = true
		}
	}
	if failed {
		t.Log(b)
		t.Log(jsonUnescaped)
		t.FailNow()
	}
}
