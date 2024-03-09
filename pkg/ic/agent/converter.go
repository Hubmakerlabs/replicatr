package agent

import (
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
		result.Since = int64(f.Since.T())
	} else {
		result.Since = -1
	}

	if f.Until != nil {
		result.Until = int64(f.Until.T())
	} else {
		result.Until = -1
	}

	if f.Limit != nil {
		result.Limit = int64(*f.Limit)
	} else {
		result.Limit = 500
	}

	return
}

func EventToCandid(e *event.T) Event {
	return Event{
		e.ID.String(),
		e.PubKey,
		int64(e.CreatedAt),
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
		CreatedAt: timestamp.T(e.CreatedAt),
		Kind:      kind.T(e.Kind),
		Tags:      t,
		Content:   e.Content,
		Sig:       e.Sig,
	}
}
