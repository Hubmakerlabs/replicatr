package agent

import (
	"github.com/aviate-labs/agent-go/candid/idl"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/eventid"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/tags"
	"mleku.dev/git/nostr/timestamp"
)

func TagMapToKV(t filter.TagMap) (keys []KeyValuePair) {
	keys = make([]KeyValuePair, 0, len(t))
	for k := range t {
		keys = append(keys, KeyValuePair{k, t[k]})
	}
	return
}

func FilterToCandid(f *filter.T) (result *Filter) {
	result = &Filter{
		IDs:     f.IDs,
		Kinds:   f.Kinds.ToUint16(),
		Authors: f.Authors,
		Tags:    TagMapToKV(f.Tags),
		Search:  f.Search,
	}
	if f.Since != nil {
		result.Since = idl.NewInt(f.Since.T().Int())
	} else {
		result.Since = idl.NewInt(-1)
	}

	if f.Until != nil {
		result.Until = idl.NewInt(f.Until.T().Int())
	} else {
		result.Until = idl.NewInt(-1)
	}

	if f.Limit != nil {
		result.Limit = idl.NewInt(*f.Limit)
	} else {
		result.Limit = idl.NewInt(500)
	}

	return
}

func EventToCandid(e *event.T) Event {
	return Event{
		e.ID.String(),
		e.PubKey,
		idl.NewInt(int(e.CreatedAt)),
		uint16(e.Kind),
		e.Tags.Slice(),
		e.Content,
		e.Sig,
	}
}

func CandidToEvent(e *Event) *event.T {
	var t tags.T
	for _, v := range e.Tags {
		t = append(t, v)
	}
	return &event.T{
		ID:        eventid.T(e.ID),
		PubKey:    e.Pubkey,
		CreatedAt: timestamp.T(e.CreatedAt.BigInt().Int64()),
		Kind:      kind.T(e.Kind),
		Tags:      t,
		Content:   e.Content,
		Sig:       e.Sig,
	}
}
