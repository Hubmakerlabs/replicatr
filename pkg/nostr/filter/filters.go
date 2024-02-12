package filter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/object"
	"mleku.online/git/slog"
)

var log, chk = slog.New(os.Stderr)

// T is a query where one or all elements can be filled in.
//
// Most of it is normal stuff but the Tags are a special case because the Go
// encode/json will not do what the specification requires, which is to unwrap
// the tag as fields.
//
//	Tags: {K1: val1, K2: val2)
//
// must be changed to
//
//	K1: val1
//	K2: val2
//
// For this reason in the original nbd-wdf/go-nostr special handling is created
// using the easyjson library that allows this '-' json tag to indicate to
// promote the key/value pairs inside to the same level of the object and not
// bundled inside another key.
//
// Because we have a native key/value type designed for ordered object JSON
// serialization we just give it special treatment in the ToObject function.
//
// The json tags are not here because they are worthless for unmarshalling and
// unnecessary for marshaling. They appear in the ToObject because all of them
// are optional fields.
//
// For the simplified handling of unmarshaling this JSON abomination this struct
// is redefined so that the Tags fields are elaborated concretely and then the
// populated tags are put into the map as they are expected to be.
type T struct {
	IDs     tag.T         `json:"ids,omitempty"`
	Kinds   kinds.T       `json:"kinds,omitempty"`
	Authors tag.T         `json:"authors,omitempty"`
	Tags    TagMap        `json:"-,omitempty"`
	Since   *timestamp.Tp `json:"since,omitempty"`
	Until   *timestamp.Tp `json:"until,omitempty"`
	Limit   *int          `json:"limit,omitempty"`
	Search  string        `json:"search,omitempty"`
}

func (f *T) ToObject() (o object.T) {
	o = object.T{
		{"ids,omitempty", f.IDs},
		{"kinds,omitempty", f.Kinds.ToArray()},
		{"authors,omitempty", f.Authors},
	}
	// these tags are not grouped under a top level key but unfolded into the
	// object, promoted to the same level as their enclosing map. Go doesn't
	// have a native "collection" type like this, but our object.T does the same
	// thing for encoding. This does also mean for this type a hand written
	// decoder will need to be written to identify and pack the keys.
	//
	// In addition, due to the nondeterministic map iteration of Go, we make a
	// temp slice and sort it.
	var tmp object.T
	for i := range f.Tags {
		key := i
		if len(i) == 1 {
			v := i[0]
			if v >= 'a' && v <= 'z' || v >= 'A' && v <= 'Z' {
				key = "#" + i
			}
		}
		tmp = append(tmp, object.KV{Key: key, Value: f.Tags[i]})
	}
	sort.Sort(tmp)
	o = append(o, tmp...)
	o = append(o, object.T{
		{"since,omitempty", f.Since},
		{"until,omitempty", f.Until},
	}...)
	o = append(o, object.KV{Key: "limit,omitempty", Value: f.Limit})
	if f.Search != "" {
		o = append(o, object.NewKV("search,omitempty", f.Search))
	}
	return
}

func (f *T) MarshalJSON() (b []byte, err error) {
	return f.ToObject().Bytes(), nil
}

// UnmarshalJSON correctly unpacks a JSON encoded T rolling up the Tags as
// they should be.
func (f *T) UnmarshalJSON(b []byte) (err error) {
	if f == nil {
		return fmt.Errorf("cannot unmarshal into nil T")
	}
	// // log.T.F("unmarshaling filter `%s`", b)
	var uf UnmarshalingFilter
	if err = json.Unmarshal(b, &uf); chk.D(err) {
		return
	}
	if err = CopyUnmarshalFilterToFilter(&uf, f); chk.D(err) {
		return
	}
	return
}

type TagMap map[string]tag.T

func (t TagMap) Clone() (t1 TagMap) {
	if t == nil {
		return
	}
	t1 = make(TagMap)
	for i := range t {
		t1[i] = t[i]
	}
	return
}

func (f *T) String() string {
	j, _ := json.Marshal(f)
	return string(j)
}

func (f *T) Matches(ev *event.T) bool {
	if ev == nil {
		// log.T.F("nil event")
		return false
	}
	if len(f.IDs) > 0 && !f.IDs.Contains(ev.ID.String()) {
		// log.T.F("no ids in filter match event\nEVENT %s\nFILTER %s", ev.ToObject().String(), f.ToObject().String())
		return false
	}
	if len(f.Kinds) > 0 && !f.Kinds.Contains(ev.Kind) {
		// log.T.F("no matching kinds in filter\nEVENT %s\nFILTER %s", ev.ToObject().String(), f.ToObject().String())
		return false
	}
	if len(f.Authors) > 0 && !f.Authors.Contains(ev.PubKey) {
		// log.T.F("no matching authors in filter\nEVENT %s\nFILTER %s", ev.ToObject().String(), f.ToObject().String())
		return false
	}
	for ff, v := range f.Tags {
		if strings.HasPrefix(ff, "#") {
			ff = ff[1:]
		}
		if len(v) > 0 && !ev.Tags.ContainsAny(ff, v) {
			// log.T.F("no matching tags in filter\nEVENT %s\nFILTER %s", ev.ToObject().String(), f.ToObject().String())
			return false
		}
		// special case for p tags
	}
	if f.Since != nil && ev.CreatedAt < timestamp.T(*f.Since) {
		// log.T.F("event is older than since\nEVENT %s\nFILTER %s", ev.ToObject().String(), f.ToObject().String())
		return false
	}
	if f.Until != nil && ev.CreatedAt > timestamp.T(*f.Until) {
		// log.T.F("event is newer than until\nEVENT %s\nFILTER %s", ev.ToObject().String(), f.ToObject().String())
		return false
	}
	return true
}

func arePointerValuesEqual[V comparable](a *V, b *V) bool {
	if a == nil && b == nil {
		return true
	}
	if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func FilterEqual(a, b *T) bool {
	// switch is a convenient way to bundle a long list of tests like this:
	switch {
	case !a.Kinds.Equals(b.Kinds),
		!a.IDs.Equals(b.IDs),
		!a.Authors.Equals(b.Authors),
		len(a.Tags) != len(b.Tags),
		!arePointerValuesEqual(a.Since, b.Since),
		!arePointerValuesEqual(a.Until, b.Until),
		a.Search != b.Search:

		return false
	}
	for f, av := range a.Tags {
		if bv, ok := b.Tags[f]; !ok {
			return false
		} else if !av.Equals(bv) {
			return false
		}
	}
	return true
}

func (f *T) Clone() (clone *T) {
	clone = &T{
		IDs:     f.IDs.Clone(),
		Authors: f.Authors.Clone(),
		Kinds:   f.Kinds.Clone(),
		Limit:   f.Limit,
		Search:  f.Search,
		Tags:    f.Tags.Clone(),
		Since:   f.Since.Clone(),
		Until:   f.Until.Clone(),
	}
	return
}
