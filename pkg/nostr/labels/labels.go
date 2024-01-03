package labels

import (
	"fmt"
)

type T = byte

// T enums for compact identification of the label.
const (
	LNil T = iota
	LEvent
	LOK
	LNotice
	LEOSE
	LClose
	LClosed
	LReq
	LCount
	LAuth
)

// List is the nip1 envelope labels, matching the above enums.
var List = EnvelopeLabel{
	LNil:    nil,
	LEvent:  []byte("EVENT"),
	LOK:     []byte("OK"),
	LNotice: []byte("NOTICE"),
	LEOSE:   []byte("EOSE"),
	LClose:  []byte("CLOSE"),
	LClosed: []byte("CLOSED"),
	LReq:    []byte("REQ"),
	LCount:  []byte("COUNT"),
	LAuth:   []byte("AUTH"),
}

type EnvelopeLabel map[T][]byte

func (l EnvelopeLabel) String() (s string) {
	s += "["
	for i := range List {
		s += fmt.Sprintf("%d:'%s',", i, List[i])
	}
	s += "]"
	return
}

// With these, labels have easy short names for the strings, as well as neat
// consistent 1 byte enum version. Having all 3 versions also makes writing the
// recogniser easier.
var (
	EVENT  = string(List[LEvent])
	OK     = string(List[LOK])
	REQ    = string(List[LReq])
	NOTICE = string(List[LNotice])
	EOSE   = string(List[LEOSE])
	CLOSE  = string(List[LClose])
	CLOSED = string(List[LClosed])
	COUNT  = string(List[LCount])
)

func GetLabel(s string) (l T) {
	for i := range List {
		if string(List[i]) == s {
			l = i
		}
	}
	return
}
