// Package object implements an ordered key/value data structure for use with
// JSON documents that must be strictly ordered in order to create a consistent
// blob of data in canonical order for creating verifiable signatures while
// delivering the data over the wire or storing it with its signature and object
// hash also present, as is used for nostr events.
//
// Rather than implementing the json.Marshal and json.Unmarshal, marshaling data
// must be done by converting the struct explicitly to a string key/interface
// value slice. See object_test.go for an example of such a function.
//
// Note that strings found in the object are automatically escaped as per
// RFC8259 with a function that avoids more than one memory allocation for the
// buffer rewrite.
package object

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

var log, chk = slog.New(os.Stderr)

type KV struct {
	Key   string
	Value interface{}
}

func NewKV(k string, v interface{}) KV { return KV{Key: k, Value: v} }

type T []KV

func (t T) Buffer(b ...*bytes.Buffer) (buf *bytes.Buffer) {
	if len(b) != 1 {
		buf = new(bytes.Buffer)
	} else {
		buf = b[0]
	}
	_, _ = fmt.Fprint(buf, "{")
	last := len(t) - 1

	var ok bool
	var str string
	var ts time.Time
	var tagsList tags.T
	for i := range t {
		// keys can have `.omitempty` after them and if present, the field is
		// omitted if it is a zero or nil value.
		var omitempty bool
		k := strings.Split(t[i].Key, ",")
		key := k[0]
		if len(k) > 1 {
			if k[1] == "omitempty" {
				omitempty = true
			}
		}
		v := t[i].Value
		// if tag of object includes omitempty and the value matches the zero of
		// the type, don't add it to the output.
		if omitempty {
			// log.T.Ln(i, len(t), t[i])
			// check for nil
			if v == nil {
				continue
			}
			switch reflect.TypeOf(v).Kind() {
			case reflect.Ptr,
				reflect.Map,
				reflect.Array,
				reflect.Chan,
				reflect.Slice:

				if reflect.ValueOf(v).IsNil() {
					continue
				}
			default:
			}
			// check for zero
			if reflect.DeepEqual(reflect.Zero(reflect.TypeOf(t[i])), t[i]) {
				continue
			}
		}

		// add the key
		_, _ = fmt.Fprint(buf, "\"", key, "\":")
		// add the value
		if str, ok = t[i].Value.(string); ok {
			v := text.EscapeJSONStringAndWrap(str)
			buf.Write(v)
		} else if reflect.TypeOf(t[i].Value).Kind() == reflect.String {
			v := text.EscapeJSONStringAndWrap(reflect.ValueOf(t[i].Value).String())
			buf.Write(v)
		} else if ts, ok = t[i].Value.(time.Time); ok {
			_, _ = fmt.Fprint(buf, ts.Unix())
		} else if tagsList, ok = t[i].Value.(tags.T); ok {
			tagsListLast := len(tagsList) - 1
			_, _ = fmt.Fprint(buf, "[")
			for tagList, te := range tagsList {
				tagLast := len(te) - 1
				_, _ = fmt.Fprint(buf, "[")
				for tagElement, t := range te {
					v := text.EscapeJSONStringAndWrap(t)
					buf.Write(v)
					if tagElement < tagLast {
						_, _ = fmt.Fprint(buf, ",")
					} else {
						_, _ = fmt.Fprint(buf, "]")
					}
				}
				if tagList < tagsListLast {
					_, _ = fmt.Fprint(buf, ",")
				}
			}
			_, _ = fmt.Fprint(buf, "]")
		} else {
			_, _ = fmt.Fprint(buf, t[i].Value)
		}
		if i < last {
			_, _ = fmt.Fprint(buf, ",")
		}
	}
	_, _ = fmt.Fprint(buf, "}")
	return buf
}

func (t T) String() string { return t.Buffer().String() }
func (t T) Bytes() []byte  { return t.Buffer().Bytes() }
func (t T) MarshalJSON() (b []byte,
	err error) {
	return t.Buffer().Bytes(), nil
}

// sort.Interface implementation

func (t T) Len() int           { return len(t) }
func (t T) Less(i, j int) bool { return t[i].Key < t[j].Key }
func (t T) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
