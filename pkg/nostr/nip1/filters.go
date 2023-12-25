package nip1

import (
	"encoding/json"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/object"
	"sort"
)

type Filters []Filter

func (eff Filters) ToArray() (a array.T) {
	for i := range eff {
		a = append(a, eff[i].ToObject())
	}
	return
}

// Filter is a query where one or all elements can be filled in.
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
type Filter struct {
	IDs     tag.T         `json:"ids,omitempty"`
	Kinds   kind.Array    `json:"kinds,omitempty"`
	Authors tag.T         `json:"authors,omitempty"`
	Tags    TagMap        `json:"-,omitempty"`
	Since   *timestamp.Tp `json:"since,omitempty"`
	Until   *timestamp.Tp `json:"until,omitempty"`
	Limit   int           `json:"limit,omitempty"`
	Search  string        `json:"search,omitempty"`
}

func (f *Filter) ToObject() (o object.T) {
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
		tmp = append(tmp, object.KV{Key: i, Value: f.Tags[i]})
	}
	sort.Sort(tmp)
	o = append(o, tmp...)
	o = append(o, object.T{
		{"since,omitempty", f.Since},
		{"until,omitempty", f.Until},
		{"limit,omitempty", f.Limit},
	}...)
	if f.Search != "" {
		o = append(o, object.NewKV("search,omitempty", f.Search))
	}
	return
}

func (f *Filter) MarshalJSON() (b []byte, e error) {
	return f.ToObject().Bytes(), nil
}

// UnmarshalJSON correctly unpacks a JSON encoded Filter rolling up the Tags as
// they should be.
func (f *Filter) UnmarshalJSON(b []byte) (e error) {
	if f == nil {
		return fmt.Errorf("cannot unmarshal into nil Filter")
	}
	uf := &UnmarshalingFilter{}
	if e = json.Unmarshal(b, uf); fails(e) {
		return
	}
	if e = CopyUnmarshalFilterToFilter(uf, f); fails(e) {
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

func (eff Filters) String() string { return eff.ToArray().String() }

func (eff Filters) Match(event *Event) bool {
	for _, filter := range eff {
		if filter.Matches(event) {
			return true
		}
	}
	return false
}

func (f *Filter) String() string {
	j, _ := json.Marshal(f)
	return string(j)
}

func (f *Filter) Matches(event *Event) bool {
	if event == nil {
		return false
	}

	if f.IDs != nil && !f.IDs.Contains(event.ID.String()) {
		return false
	}

	if f.Kinds != nil && !f.Kinds.Contains(event.Kind) {
		return false
	}

	if f.Authors != nil && !f.Authors.Contains(event.PubKey) {
		return false
	}

	for f, v := range f.Tags {
		if v != nil && !event.Tags.ContainsAny(f, v) {
			return false
		}
	}

	if f.Since != nil && event.CreatedAt < timestamp.T(*f.Since) {
		return false
	}

	if f.Until != nil && event.CreatedAt > timestamp.T(*f.Until) {
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

func FilterEqual(a, b *Filter) bool {
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

func (f *Filter) Clone() (clone *Filter) {
	clone = &Filter{
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
