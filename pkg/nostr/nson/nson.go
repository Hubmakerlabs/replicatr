package nson

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"mleku.dev/git/slog"
)

/*
	nson size
	  kind chars
	    content chars
	        number of tags (let's say it's two)
	          number of items on the first tag (let's say it's three)
	            number of chars on the first item
	                number of chars on the second item
	                    number of chars on the third item
	                        number of items on the second tag (let's say it's two)
	                          number of chars on the first item
	                              number of chars on the second item

"nson":"xxkkccccttnn111122223333nn11112222"
*/

var log, chk = slog.New(os.Stderr)

const (
	IdStart        = 7
	IdEnd          = 7 + 64
	PubkeyStart    = 83
	PubkeyEnd      = 83 + 64
	SigStart       = 156
	SigEnd         = 156 + 128
	CreatedAtStart = 299
	CreatedAtEnd   = 299 + 10

	StringStart = 318     // the actual json string for the "nson" field
	ValuesStart = 318 + 2 // skipping the first byte which delimits the nson size

	MarkerStart = 309 // this is used just to determine if an event is nson or not
	MarkerEnd   = 317 // it's just the `,"nson":` (including ,": garbage to reduce false positives) part
)

var ErrNotNSON = fmt.Errorf("not nson")

func UnmarshalBytes(data []byte, evt *event.T) (err error) {
	return Unmarshal(unsafe.String(unsafe.SliceData(data), len(data)), evt)
}

// Unmarshal turns a NSON string into a event.T struct.
func Unmarshal(data string, evt *event.T) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = log.E.Err("failed to decode nson: %v", r)
		}
	}()

	// check if it's nson
	if data[MarkerStart:MarkerEnd] != ",\"nson\":" {
		return ErrNotNSON
	}

	// nson values
	nsonSize, nsonDescriptors := parseDescriptors(data)

	// static fields
	evt.ID = eventid.T(data[IdStart:IdEnd])
	evt.PubKey = data[PubkeyStart:PubkeyEnd]
	evt.Sig = data[SigStart:SigEnd]
	ts, _ := strconv.ParseUint(data[CreatedAtStart:CreatedAtEnd], 10, 64)
	evt.CreatedAt = timestamp.T(ts)

	// dynamic fields
	// kind
	kindChars := int(nsonDescriptors[0])
	kindStart := ValuesStart + nsonSize + 9 // len(`","kind":`)
	ki, _ := strconv.Atoi(data[kindStart : kindStart+kindChars])
	evt.Kind = kind.T(ki)
	// content
	contentChars := int(binary.BigEndian.Uint16(nsonDescriptors[1:3]))
	contentStart := kindStart + kindChars + 12 // len(`,"content":"`)
	evt.Content, _ = strconv.Unquote(data[contentStart-1 : contentStart+contentChars+1])

	// tags
	nTags := int(nsonDescriptors[3])
	evt.Tags = make(tags.T, nTags)
	tagsStart := contentStart + contentChars + 9 // len(`","tags":`)

	nsonIndex := 3
	tagsIndex := tagsStart
	for t := 0; t < nTags; t++ {
		nsonIndex++
		tagsIndex += 1 // len(`[`) or len(`,`)
		nItems := int(nsonDescriptors[nsonIndex])
		tg := make(tag.T, nItems)
		for n := 0; n < nItems; n++ {
			nsonIndex++
			itemStart := tagsIndex + 2 // len(`["`) or len(`,"`)
			itemChars := int(binary.BigEndian.Uint16(nsonDescriptors[nsonIndex:]))
			nsonIndex++
			tg[n], _ = strconv.Unquote(data[itemStart-1 : itemStart+itemChars+1])
			tagsIndex = itemStart + itemChars + 1 // len(`"`)
		}
		tagsIndex += 1 // len(`]`)
		evt.Tags[t] = tg
	}

	return err
}

func MarshalBytes(evt *event.T) ([]byte, error) {
	v, err := Marshal(evt)
	return unsafe.Slice(unsafe.StringData(v), len(v)), err
}

func Marshal(evt *event.T) (string, error) {
	// start building the nson descriptors (without the first byte that represents the nson size)
	nsonBuf := make([]byte, 256)

	// build the tags
	nTags := len(evt.Tags)
	nsonBuf[3] = uint8(nTags)
	nsonIndex := 3 // start here

	tagBuilder := strings.Builder{}
	tagBuilder.Grow(1000) // a guess
	tagBuilder.WriteString(`[`)
	for t, tag := range evt.Tags {
		nItems := len(tag)
		nsonIndex++
		nsonBuf[nsonIndex] = uint8(nItems)

		tagBuilder.WriteString(`[`)
		for i, item := range tag {
			v := strconv.Quote(item)
			nsonIndex++
			binary.BigEndian.PutUint16(nsonBuf[nsonIndex:], uint16(len(v)-2))
			nsonIndex++
			tagBuilder.WriteString(v)
			if nItems > i+1 {
				tagBuilder.WriteString(`,`)
			}
		}
		tagBuilder.WriteString(`]`)
		if nTags > t+1 {
			tagBuilder.WriteString(`,`)
		}
	}
	tagBuilder.WriteString(`]}`)
	nsonBuf = nsonBuf[0 : nsonIndex+1]

	ki := strconv.Itoa(int(evt.Kind))
	kindChars := len(ki)
	nsonBuf[0] = uint8(kindChars)

	content := strconv.Quote(evt.Content)
	contentChars := len(content) - 2
	binary.BigEndian.PutUint16(nsonBuf[1:3], uint16(contentChars))

	// actually build the json
	base := strings.Builder{}
	base.Grow(ValuesStart + // everything up to "nson":
		2 + len(nsonBuf)*2 + // nson
		9 + kindChars + // kind and its label
		12 + contentChars + // content and its label
		9 + tagBuilder.Len() + // tags and its label
		2, // the end
	)
	base.WriteString(`{"id":"` + evt.ID.String() + `","pubkey":"` + evt.PubKey + `","sig":"` + evt.Sig +
		`","created_at":` + strconv.FormatInt(int64(evt.CreatedAt), 10) + `,"nson":"`)

	nsonSizeBytes := len(nsonBuf)
	if nsonSizeBytes > 255 {
		return "", fmt.Errorf("can't encode to nson, there are too many tags or tag items")
	}
	base.WriteString(hex.EncodeToString([]byte{uint8(nsonSizeBytes)})) // nson size (bytes)

	base.WriteString(hex.EncodeToString(nsonBuf)) // nson descriptors
	base.WriteString(`","kind":` + ki + `,"content":` + content + `,"tags":`)
	base.WriteString(tagBuilder.String() /* includes the end */)

	return base.String(), nil
}

func parseDescriptors(data string) (int, []byte) {
	nsonSizeBytes, _ := hex.DecodeString(data[StringStart:ValuesStart])
	// number of bytes is given, we x2 because the string is in hex
	size := int(nsonSizeBytes[0]) * 2
	values, _ := hex.DecodeString(data[ValuesStart : ValuesStart+size])
	return size, values
}

// An Event is basically a wrapper over the string that makes it easy to get
// each event property (except tags).
type Event struct {
	data            string
	descriptorsSize int
	descriptors     []byte
}

func New(nsonText string) Event {
	return Event{data: nsonText}
}

func (ne *Event) parseDescriptors() {
	if ne.descriptors == nil {
		ne.descriptorsSize, ne.descriptors = parseDescriptors(ne.data)
	}
}

func (ne *Event) GetID() string     { return ne.data[IdStart:IdEnd] }
func (ne *Event) GetPubkey() string { return ne.data[PubkeyStart:PubkeyEnd] }
func (ne *Event) GetSig() string    { return ne.data[SigStart:SigEnd] }
func (ne *Event) GetCreatedAt() timestamp.T {
	ts, _ := strconv.ParseUint(ne.data[CreatedAtStart:CreatedAtEnd], 10, 64)
	return timestamp.T(ts)
}

func (ne *Event) GetKind() int {
	ne.parseDescriptors()

	kindChars := int(ne.descriptors[0])
	kindStart := ValuesStart + ne.descriptorsSize + 9 // len(`","kind":`)
	kind, _ := strconv.Atoi(ne.data[kindStart : kindStart+kindChars])

	return kind
}

func (ne *Event) GetContent() string {
	ne.parseDescriptors()

	kindChars := int(ne.descriptors[0])
	kindStart := ValuesStart + ne.descriptorsSize + 9 // len(`","kind":`)

	contentChars := int(binary.BigEndian.Uint16(ne.descriptors[1:3]))
	contentStart := kindStart + kindChars + 12 // len(`,"content":"`)
	content, _ := strconv.Unquote(`"` + ne.data[contentStart:contentStart+contentChars] + `"`)

	return content
}
