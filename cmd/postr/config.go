package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip4"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/fatih/color"
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
	Relays        map[string]*RelayPerms
	Emojis        map[string]string
	Checklist     map[string]struct{}
	RelayIterator func(context.T, *relay.Relay) bool
)

// C is the configuration for the client
type C struct {
	Relays    `json:"relays"`
	Follows   `json:"follows"`
	SecretKey string    `json:"privatekey"`
	Updated   time.Time `json:"updated"`
	Emojis    `json:"emojis"`
	NwcURI    string `json:"nwc-uri"`
	NwcPub    string `json:"nwc-pub"`
	verbose   bool
	tempRelay bool
	sk        string
}

// LastUpdated returns whether there was an update in the most recent time
// duration previous to the current time.
func (cfg *C) LastUpdated(t time.Duration) bool {
	return cfg.Updated.Add(t).Before(time.Now())
}

// Touch sets the last updated time of the configuration to the current time.
func (cfg *C) Touch() { cfg.Updated = time.Now() }

// GetFollows is
func (cfg *C) GetFollows(profile string) (profiles Follows, e error) {
	var mu sync.Mutex
	var pub string
	if pub, _, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	log.D.Ln("pub", pub)
	// get followers
	if (cfg.LastUpdated(3*time.Hour) && !cfg.tempRelay) ||
		len(cfg.Follows) == 0 {

		mu.Lock()
		cfg.Follows = make(Follows)
		mu.Unlock()
		m := make(Checklist)
		cfg.Do(readPerms, cfg.GetRelaysAndTags(pub, &m, &mu))
		log.D.F("found %d followers", len(m))
		if len(m) > 0 {
			var follows []string
			for k := range m {
				follows = append(follows, k)
			}
			for i := 0; i < len(follows); i += 500 {
				// Calculate the end index based on the current index and slice
				// length
				end := i + 500
				if end > len(follows) {
					end = len(follows)
				}
				// get follower's descriptions
				cfg.Do(readPerms, cfg.PopulateFollows(&follows, &i, &end, &mu))
			}
		}
		cfg.Touch()
		if e = cfg.save(profile); log.Fail(e) {
			return nil, e
		}
	}
	return cfg.Follows, nil
}
func (cfg *C) GetRelaysAndTags(pub string, m *Checklist,
	mu *sync.Mutex) RelayIterator {
	return func(c context.T, rl *relay.Relay) bool {
		evs, e := rl.QuerySync(c, &filter.T{
			Kinds:   kinds.T{kind.ContactList},
			Authors: []string{pub},
			Limit:   1,
		})
		if log.Fail(e) {
			return true
		}
		log.D.Ln("evs", len(evs[0].Tags))
		for _, ev := range evs {
			var rm Relays
			if cfg.tempRelay == false {
				if e = json.Unmarshal([]byte(ev.Content), &rm); log.Fail(e) {
					// continue
				} else {
					for k, v1 := range cfg.Relays {
						if v2, ok := rm[k]; ok {
							v2.Search = v1.Search
						}
					}
					cfg.Relays = rm
				}
			}
			log.D.S(ev.Tags)
			for _, tag := range ev.Tags {
				if len(tag) >= 2 && tag[0] == "p" {
					log.D.Ln("p tag", tag.Relay(), tag.Key(), tag.Value())
					mu.Lock()
					(*m)[tag[1]] = struct{}{}
					mu.Unlock()
				}
			}
			mu.Lock()
			log.D.S(*m)
			mu.Unlock()

		}
		return true
	}
}

func (cfg *C) PopulateFollows(f *[]string, i, end *int,
	mu *sync.Mutex) RelayIterator {
	return func(c context.T, rl *relay.Relay) bool {
		evs, e := rl.QuerySync(c, &filter.T{
			Kinds:   kinds.T{kind.ProfileMetadata},
			Authors: (*f)[*i:*end], // Use the updated end index
		})
		if log.Fail(e) {
			return true
		}
		for _, ev := range evs {
			p := &Metadata{}
			e = json.Unmarshal([]byte(ev.Content), p)
			if e == nil {
				mu.Lock()
				cfg.Follows[ev.PubKey] = p
				mu.Unlock()
			}
		}
		return true
	}
}

// FindRelay is
func (cfg *C) FindRelay(c context.T, r *RelayPerms) *relay.Relay {
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
		rl, e := relay.RelayConnect(c, k)
		if log.Fail(e) {
			continue
		}
		return rl
	}
	return nil
}

// Do runs a query on all of the configured relays. Return false in the closure
// to end the iteration.
func (cfg *C) Do(r *RelayPerms, f RelayIterator) {
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
		go func(wg *sync.WaitGroup, k string, v *RelayPerms) {
			defer wg.Done()
			rl, e := relay.RelayConnect(c, k)
			if log.Fail(e) {
				log.D.Ln(e)
				return
			}
			if !f(c, rl) {
				c.Done()
			}
			log.Fail(rl.Close())
		}(&wg, k, v)
	}
	wg.Wait()
}

func (cfg *C) save(profile string) (e error) {
	if cfg.tempRelay {
		return nil
	}
	dir, e := configDir()
	if log.Fail(e) {
		return e
	}
	dir = filepath.Join(dir, appName)

	var fp string
	if profile == "" {
		fp = filepath.Join(dir, "config.json")
	} else {
		fp = filepath.Join(dir, "config-"+profile+".json")
	}
	b, e := json.MarshalIndent(&cfg, "", "  ")
	if log.Fail(e) {
		return e
	}
	return os.WriteFile(fp, b, 0644)
}

// Decode is
func (cfg *C) Decode(ev *event.T) (e error) {
	var sk string
	var pub string
	if pub, _, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
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
	ss, e := nip4.ComputeSharedSecret(sp, sk)
	if log.Fail(e) {
		return e
	}
	content, e := nip4.Decrypt(ev.Content, ss)
	if log.Fail(e) {
		return e
	}
	ev.Content = string(content)
	return nil
}

// PrintEvents is
func (cfg *C) PrintEvents(evs []*event.T, f Follows, asJson, extra bool) {
	if asJson {
		// if extra {
		var events []Event
		for _, ev := range evs {
			if profile, ok := f[ev.PubKey]; ok {
				events = append(events, Event{
					Event:   ev,
					Profile: profile,
				})
			}
		}
		for _, ev := range events {
			log.Fail(json.NewEncoder(os.Stdout).Encode(ev))
		}
		// } else {
		// 	for _, ev := range evs {
		// 		log.Fail(json.NewEncoder(os.Stdout).Encode(ev))
		// 	}
		// }
		return
	}

	buf := make([]byte, 4096)
	buffer := bytes.NewBuffer(buf)
	fgHiRed := color.New(color.FgHiRed)
	fgRed := color.New(color.FgRed)
	fgNormal := color.New(color.Reset)
	fgHiBlue := color.Set(color.FgHiBlue)
	for _, ev := range evs {
		profile, ok := f[ev.PubKey]
		if ok {
			color.Set(color.FgHiRed)
			fgHiRed.Fprint(buffer, profile.Name, " ")
			fgHiBlue.Fprintln(buffer, ev.CreatedAt.Time())
			fgRed.Fprint(buffer, "pubkey ")
			fgRed.Fprint(buffer, ev.PubKey)
			fgHiBlue.Fprint(buffer, " note ID: ")
			fgHiBlue.Fprintln(buffer, ev.ID)
			fgNormal.Fprintln(buffer, ev.Content)
		} else {
			fgRed.Fprint(buffer, "pubkey ")
			fgRed.Fprint(buffer, ev.PubKey)
			fgHiBlue.Fprint(buffer, " note ID: ")
			fgHiBlue.Fprintln(buffer, ev.ID)
			fgHiBlue.Fprintln(buffer, ev.CreatedAt.Time())
			fgNormal.Fprintln(buffer, ev.Content)
		}
		fgNormal.Fprintln(buffer)
	}
	fmt.Println(buffer.String())
}

func (cfg *C) GetEvents(ids []string) (evs []*event.T) {
	cfg.Do(readPerms, func(c context.T, rl *relay.Relay) bool {
		events, e := rl.QuerySync(c, &filter.T{
			IDs:   ids,
			Kinds: kinds.T{kind.TextNote},
			Limit: len(ids),
		})
		if log.Fail(e) {
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
	var mu sync.Mutex
	found := false
	var m sync.Map
	cfg.Do(readPerms, func(c context.T, rl *relay.Relay) bool {
		mu.Lock()
		if found {
			mu.Unlock()
			return false
		}
		mu.Unlock()
		evs, e := rl.QuerySync(c, &f)
		if log.Fail(e) {
			return true
		}
		log.D.Ln("number of events found", len(evs))
		for _, ev := range evs {
			if _, ok := m.Load(ev.ID); !ok {
				if ev.Kind == kind.EncryptedDirectMessage {
					if e = cfg.Decode(ev); log.Fail(e) {
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
	// m.Range(func(key any, value any) bool {
	// 	log.D.Ln("event ID", key.(eventid.EventID).String())
	// 	log.D.Ln(value.(*event.T).ToObject().String())
	// 	return true
	// })
	var evs []*event.T
	m.Range(func(k, v any) bool {
		evs = append(evs, v.(*event.T))
		return true
	})
	sort.Slice(evs, func(i, j int) bool {
		return evs[i].CreatedAt < evs[j].CreatedAt
	})
	return evs
}

// ZapInfo is
func (cfg *C) ZapInfo(pub string) (*Lnurlp, error) {
	rl := cfg.FindRelay(context.Bg(), readPerms)
	if rl == nil {
		return nil, errors.New("cannot connect relays")
	}
	defer rl.Close()
	// get set-metadata
	f := filter.T{
		Kinds:   kinds.T{kind.ProfileMetadata},
		Authors: []string{pub},
		Limit:   1,
	}
	evs := cfg.Events(f)
	if len(evs) == 0 {
		return nil, errors.New("cannot find user")
	}
	var profile Metadata
	e := json.Unmarshal([]byte(evs[0].Content), &profile)
	if log.Fail(e) {
		return nil, e
	}
	tok := strings.SplitN(profile.Lud16, "@", 2)
	if log.Fail(e) {
		return nil, e
	}
	if len(tok) != 2 {
		return nil, errors.New("receipt address is not valid")
	}
	var resp *http.Response
	resp, e = http.Get("https://" + tok[1] + "/.well-known/lnurlp/" + tok[0])
	if log.Fail(e) {
		return nil, e
	}
	defer log.Fail(resp.Body.Close())

	var lp Lnurlp
	if e = json.NewDecoder(resp.Body).Decode(&lp); log.Fail(e) {
		return nil, e
	}
	return &lp, nil
}
