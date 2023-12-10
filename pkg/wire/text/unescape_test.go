package text

import (
	"fmt"
	log2 "mleku.online/git/log"
	"testing"
)

func TestUnescapeByteString(t *testing.T) {
	log2.SetLogLevel(log2.Debug)
	b := make([]byte, 256)
	for i := range b {
		b[i] = byte(i)
	}
	escaped := make([]byte, 256*6)
	for i := range b {
		copy(escaped[i*6:i*6+6], fmt.Sprintf("\\u00%02x", i))
	}
	unescaped := UnescapeByteString(escaped)
	log.D.Ln(len(unescaped), string(unescaped))
}
