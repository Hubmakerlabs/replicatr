package nip1

import (
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

// NoticeEnvelope is a relay message intended to be shown to users in a nostr
// client interface.
type NoticeEnvelope struct {
	Text string
}

func NewNoticeEnvelope(text string) (E *NoticeEnvelope) {
	E = &NoticeEnvelope{Text: text}
	return
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *NoticeEnvelope) Label() (l Label) { return LNotice }

func (E *NoticeEnvelope) ToArray() (a array.T) {
	return array.T{NOTICE, E.Text}
}

func (E *NoticeEnvelope) String() (s string) {
	return E.ToArray().String()
}

func (E *NoticeEnvelope) Bytes() (s []byte) {
	return E.ToArray().Bytes()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *NoticeEnvelope) MarshalJSON() (bytes []byte, e error) {
	bytes = E.ToArray().Bytes()
	return
}

// Unmarshal the envelope.
func (E *NoticeEnvelope) Unmarshal(buf *text.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label (note we aren't checking that only
	// whitespace intervenes because laziness, usually this is the very next
	// character).
	if e = buf.ScanUntil(','); e != nil {
		return
	}
	// Next character we find will be open quotes for the notice text.
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var noticeText []byte
	// read the string
	if noticeText, e = buf.ReadUntil('"'); fails(e) {
		return fmt.Errorf("unterminated quotes in JSON, probably truncated read")
	}
	E.Text = string(text.UnescapeByteString(noticeText))
	// log.D.F("'%s'", E.Text)
	return
}
