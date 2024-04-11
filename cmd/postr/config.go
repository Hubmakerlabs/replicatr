package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/crypt"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
)

// RelayPerms is
type RelayPerms struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Search bool `json:"search"`
}

var readPerms = &RelayPerms{Read: true}
var writePerms = &RelayPerms{Write: true}

// Event is
type Event struct {
	Event   *event.T  `json:"event"`
	Profile *Metadata `json:"profile"`
}

// Metadata is
type Metadata struct {
	Name        string `json:"name"`
	Website     string `json:"website"`
	Nip05       string `json:"nip05"`
	Picture     string `json:"picture"`
	Banner      string `json:"banner"`
	Lud16       string `json:"lud16"`
	DisplayName string `json:"display_name"`
	About       string `json:"about"`
}

type (
	Follows       map[string]*Metadata
	FollowsRelays map[string][]string
	Relays        map[string]*RelayPerms
	Emojis        map[string]string
	Checklist     map[string]struct{}
	RelayIter     func(context.T, *client.T) bool
)

// C is the configuration for the client
type C struct {
	Relays         Relays        `json:"relays"`
	Follows        Follows       `json:"follows"`
	FollowsRelays  FollowsRelays `json:"follows_relays"`
	SecretKey      string        `json:"secretkey"`
	Updated        time.Time     `json:"updated"`
	Emojis         `json:"emojis"`
	NwcURI         string `json:"nwc-uri"`
	NwcPub         string `json:"nwc-pub"`
	EventURLPrefix string `json:"nevent-url"`
	verbose        bool
	trace          bool
	tempRelay      bool
	sk             string
	sync.Mutex
}

// LastUpdated returns whether there was an update in the most recent time
// duration previous to the current time.
func (cfg *C) LastUpdated(t time.Duration) bool {
	return cfg.Updated.Add(t).Before(time.Now())
}

// Touch sets the last updated time of the configuration to the current time.
func (cfg *C) Touch() { cfg.Updated = time.Now() }

// FindRelay is
func (cfg *C) FindRelay(c context.T, r *RelayPerms) *client.T {
	for k, v := range cfg.Relays {
		if r.Write && !v.Write {
			continue
		}
		if !cfg.tempRelay && r.Search && !v.Search {
			continue
		}
		if !r.Write && !v.Read {
			continue
		}
		log.D.F("trying relay: %s", k)
		rl, err := client.Connect(c, k)
		if chk.D(err) {
			continue
		}
		return rl
	}
	return nil
}

// Do runs a query on all of the configured relays. Return false in the closure
// to end the iteration.
func (cfg *C) Do(r *RelayPerms, f RelayIter) {
	var wg sync.WaitGroup
	c := context.Bg()
	for k, v := range cfg.Relays {
		if r.Write && !v.Write {
			continue
		}
		if r.Search && !v.Search {
			continue
		}
		if !r.Write && !v.Read {
			continue
		}
		wg.Add(1)
		log.D.Ln("running iterator on", k, v)
		go func(wg *sync.WaitGroup, k string, v *RelayPerms) {
			defer wg.Done()
			log.D.Ln("connecting to relay", k)
			rl, err := client.Connect(c, k)
			if chk.D(err) {
				return
			}
			if !f(c, rl) {
				c.Done()
			}
			chk.D(rl.Close())
		}(&wg, k, v)
	}
	log.D.Ln("waiting for iterators to finish")
	wg.Wait()
}

// Decode is
func (cfg *C) Decode(ev *event.T) (err error) {
	var sk string
	var pub string
	if pub, _, err = getPubFromSec(cfg.SecretKey); chk.D(err) {
		return
	}
	tag := ev.Tags.GetFirst([]string{"p"})
	if tag == nil {
		return errors.New("is not author")
	}
	sp := tag.Value()
	if sp != pub {
		if ev.PubKey != pub {
			return errors.New("is not author")
		}
	} else {
		sp = ev.PubKey
	}
	ss, err := crypt.ComputeSharedSecret(sk, sp)
	if chk.D(err) {
		return err
	}
	content, err := crypt.Decrypt(ev.Content, ss)
	if chk.D(err) {
		return err
	}
	ev.Content = string(content)
	return nil
}

func (cfg *C) GetEvents(ids []string) (evs []*event.T) {
	cfg.Do(readPerms, func(c context.T, rl *client.T) bool {
		limit := len(ids)
		events, err := rl.QuerySync(c, &filter.T{
			IDs:   ids,
			Kinds: kinds.T{kind.TextNote},
			Limit: &limit,
		})
		if chk.D(err) {
			return false
		}
		evs = append(evs, events...)
		return true
	})
	return
}

// Events queries for a set of events based on a filter and returns a slice of
// events that were returned by the relay.
func (cfg *C) Events(f filter.T) []*event.T {
	log.D.Ln("getting events")
	var mu sync.Mutex
	found := false
	var m sync.Map
	cfg.Do(readPerms, func(c context.T, rl *client.T) bool {
		mu.Lock()
		if found {
			mu.Unlock()
			return false
		}
		mu.Unlock()
		evs, err := rl.QuerySync(c, &f)
		if chk.D(err) {
			return true
		}
		log.D.Ln("number of events found", len(evs))
		for _, ev := range evs {
			if _, ok := m.Load(ev.ID); !ok {
				if ev.Kind == kind.EncryptedDirectMessage {
					if err = cfg.Decode(ev); chk.D(err) {
						continue
					}
				}
				m.LoadOrStore(ev.ID, ev)
				if len(f.IDs) == 1 {
					mu.Lock()
					found = true
					c.Done()
					mu.Unlock()
					break
				}
			}
		}
		return true
	})
	m.Range(func(key any, value any) bool {
		log.D.Ln("event ID", key.(eventid.T).String())
		log.D.Ln(value.(*event.T).ToObject().String())
		return true
	})
	var evs []*event.T
	m.Range(func(k, v any) bool {
		evs = append(evs, v.(*event.T))
		return true
	})
	sort.Slice(evs, func(i, j int) bool {
		return evs[i].CreatedAt < evs[j].CreatedAt
	})
	log.D.Ln("got events?", len(evs))
	return evs
}

var one = 1

// ZapInfo is
func (cfg *C) ZapInfo(pub string) (*Lnurlp, error) {
	rl := cfg.FindRelay(context.Bg(), readPerms)
	if rl == nil {
		return nil, errors.New("cannot connect relays")
	}
	defer chk.E(rl.Close())
	// get set-metadata

	f := filter.T{
		Kinds:   kinds.T{kind.ProfileMetadata},
		Authors: []string{pub},
		Limit:   &one,
	}
	evs := cfg.Events(f)
	if len(evs) == 0 {
		return nil, errors.New("cannot find user")
	}
	var profile Metadata
	err := json.Unmarshal([]byte(evs[0].Content), &profile)
	if chk.D(err) {
		return nil, err
	}
	tok := strings.SplitN(profile.Lud16, "@", 2)
	if chk.D(err) {
		return nil, err
	}
	if len(tok) != 2 {
		return nil, errors.New("receipt address is not valid")
	}
	var resp *http.Response
	resp, err = http.Get("https://" + tok[1] + "/.well-known/lnurlp/" + tok[0])
	if chk.D(err) {
		return nil, err
	}
	defer chk.D(resp.Body.Close())

	var lp Lnurlp
	if err = json.NewDecoder(resp.Body).Decode(&lp); chk.D(err) {
		return nil, err
	}
	return &lp, nil
}
