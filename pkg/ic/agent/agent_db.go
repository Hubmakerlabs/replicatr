package agent

import (
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
)

func (b *Backend) QueryEvents(f *filter.T) (ch event.C, err error) {
	ch = make(event.C)
	go func() {
		if f == nil {
			err = log.E.Err("nil filter for query")
			return
		}
		var candidEvents []Event
		if candidEvents, err = b.GetCandidEvent(FilterToCandid(f)); err != nil {
			split := strings.Split(err.Error(), "Error: ")
			if len(split) == 2 && split[1] != "No events found" {
				log.E.F("IC error: %s", split[1])
			}
			return
		}
		log.I.Ln("got", len(candidEvents), "events")
		for i, e := range candidEvents {
			select {
			case <-b.Ctx.Done():
				return
			default:
			}
			log.T.Ln("sending event", i)
			ch <- CandidToEvent(&e)
		}
		log.T.Ln("done sending events")
	}()
	return
}

func (b *Backend) SaveEvent(e *event.T) (err error) {
	select {
	case <-b.Ctx.Done():
		return
	default:
	}

	if err = b.SaveCandidEvent(EventToCandid(e)); chk.E(err) {
		return
	}
	return
}

// DeleteEvent deletes an event matching the given event.
// todo: not yet implemented, but there is already a backend function for this
func (b *Backend) DeleteEvent(ev *event.T) (err error) {
	select {
	case <-b.Ctx.Done():
		return
	default:
	}
	if err = b.DeleteCandidEvent(EventToCandid(ev)); chk.E(err) {
		return
	}
	return
}

// CountEvents counts how many events match the filter in the IC.
// todo: use the proper count events API call in the canister
func (b *Backend) CountEvents(f *filter.T) (count int, err error) {
	if f == nil {
		err = log.E.Err("nil filter for count query")
		return
	}
	count, err = b.CountCandidEvent(FilterToCandid(f))
	return
}

func (b *Backend) ClearEvents() (err error) {
	err = b.ClearCandidEvents()
	return
}
