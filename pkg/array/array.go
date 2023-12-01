// Package array implements an interface slice data structure that produces JSON
// as its string/byte string output.
//
// Note that strings found in the object are automatically escaped as per
// RFC8259 with a function that avoids more than one memory allocation for the
// buffer rewrite.
package array

import (
	"bytes"
	"fmt"
	"mleku.online/git/replicatr/pkg/jsontext"
	"time"
)

type T []interface{}

func (t T) String() string {
	return t.Buffer().String()
}

func (t T) Bytes() []byte {
	return t.Buffer().Bytes()
}

func (t T) Buffer() *bytes.Buffer {
	buf := new(bytes.Buffer)
	_, _ = fmt.Fprint(buf, "[")
	last := len(t) - 1
	var ok bool
	var str string
	var ts time.Time
	for i := range t {
		if str, ok = t[i].(string); ok {
			buf.Write(jsontext.EscapeJSONStringAndWrap(str))
		} else if ts, ok = t[i].(time.Time); ok {
			_, _ = fmt.Fprint(buf, ts.Unix())
		} else {
			_, _ = fmt.Fprint(buf, t[i])
		}
		if i != last {
			_, _ = fmt.Fprint(buf, ",")
		}
	}
	_, _ = fmt.Fprint(buf, "]")
	return buf
}
