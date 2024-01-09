package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip04"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/urfave/cli/v2"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
	"github.com/fatih/color"
)

var log = log2.GetStd()

const name = "algia"

const version = "0.0.54"

var revision = "HEAD"

// RelayPerms is
type RelayPerms struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Search bool `json:"search"`
}

// Config is
type Config struct {
	Relays     map[string]RelayPerms `json:"relays"`
	Follows    map[string]Profile    `json:"follows"`
	PrivateKey string                `json:"privatekey"`
	Updated    time.Time             `json:"updated"`
	Emojis     map[string]string     `json:"emojis"`
	NwcURI     string                `json:"nwc-uri"`
	NwcPub     string                `json:"nwc-pub"`
	verbose    bool
	tempRelay  bool
	sk         string
}

// Event is
type Event struct {
	Event   *event.T `json:"event"`
	Profile Profile  `json:"profile"`
}

// Profile is
type Profile struct {
	Website     string `json:"website"`
	Nip05       string `json:"nip05"`
	Picture     string `json:"picture"`
	Lud16       string `json:"lud16"`
	DisplayName string `json:"display_name"`
	About       string `json:"about"`
	Name        string `json:"name"`
}

func configDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		dir, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		return filepath.Join(dir, ".config"), nil
	default:
		return os.UserConfigDir()
	}
}

func loadConfig(profile string) (*Config, error) {
	dir, e := configDir()
	if e != nil {
		return nil, e
	}
	dir = filepath.Join(dir, "algia")

	var fp string
	if profile == "" {
		fp = filepath.Join(dir, "config.json")
	} else if profile == "?" {
		names, e := filepath.Glob(filepath.Join(dir, "config-*.json"))
		if e != nil {
			return nil, e
		}
		for _, name := range names {
			name = filepath.Base(name)
			name = strings.TrimLeft(name[6:len(name)-5], "-")
			fmt.Println(name)
		}
		os.Exit(0)
	} else {
		fp = filepath.Join(dir, "config-"+profile+".json")
	}
	os.MkdirAll(filepath.Dir(fp), 0700)

	b, e := ioutil.ReadFile(fp)
	if e != nil {
		return nil, e
	}
	var cfg Config
	e = json.Unmarshal(b, &cfg)
	if e != nil {
		return nil, e
	}
	if len(cfg.Relays) == 0 {
		cfg.Relays = map[string]RelayPerms{}
		cfg.Relays["wss://relay.nostr.band"] = RelayPerms{
			Read:   true,
			Write:  true,
			Search: true,
		}
	}
	return &cfg, nil
}

// GetFollows is
func (cfg *Config) GetFollows(profile string) (map[string]Profile, error) {
	var mu sync.Mutex
	var pub string
	if _, s, e := nip19.Decode(cfg.PrivateKey); e == nil {
		if pub, e = keys.GetPublicKey(s.(string)); e != nil {
			return nil, e
		}
	} else {
		return nil, e
	}

	// get followers
	if (cfg.Updated.Add(3*time.Hour).Before(time.Now()) && !cfg.tempRelay) || len(cfg.Follows) == 0 {
		mu.Lock()
		cfg.Follows = map[string]Profile{}
		mu.Unlock()
		m := map[string]struct{}{}

		cfg.Do(RelayPerms{Read: true}, func(ctx context.T, rl *relays.Relay) bool {
			evs, e := rl.QuerySync(ctx, filter.T{Kinds: []int{event.KindContactList}, Authors: []string{pub}, Limit: 1})
			if e != nil {
				return true
			}
			for _, ev := range evs {
				var rm map[string]RelayPerms
				if cfg.tempRelay == false {
					if e := json.Unmarshal([]byte(ev.Content), &rm); e == nil {
						for k, v1 := range cfg.Relays {
							if v2, ok := rm[k]; ok {
								v2.Search = v1.Search
							}
						}
						cfg.Relays = rm
					}
				}
				for _, tag := range ev.Tags {
					if len(tag) >= 2 && tag[0] == "p" {
						mu.Lock()
						m[tag[1]] = struct{}{}
						mu.Unlock()
					}
				}
			}
			return true
		})
		if cfg.verbose {
			fmt.Printf("found %d followers\n", len(m))
		}
		if len(m) > 0 {
			follows := []string{}
			for k := range m {
				follows = append(follows, k)
			}

			for i := 0; i < len(follows); i += 500 {
				// Calculate the end index based on the current index and slice length
				end := i + 500
				if end > len(follows) {
					end = len(follows)
				}

				// get follower's descriptions
				cfg.Do(RelayPerms{Read: true}, func(ctx context.T, rl *relays.Relay) bool {
					evs, e := rl.QuerySync(ctx, filter.T{
						Kinds:   []int{event.KindProfileMetadata},
						Authors: follows[i:end], // Use the updated end index
					})
					if e != nil {
						return true
					}
					for _, ev := range evs {
						var profile Profile
						e := json.Unmarshal([]byte(ev.Content), &profile)
						if e == nil {
							mu.Lock()
							cfg.Follows[ev.PubKey] = profile
							mu.Unlock()
						}
					}
					return true
				})
			}
		}

		cfg.Updated = time.Now()
		if e := cfg.save(profile); e != nil {
			return nil, e
		}
	}
	return cfg.Follows, nil
}

// FindRelay is
func (cfg *Config) FindRelay(ctx context.T, r RelayPerms) *relays.Relay {
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
		if cfg.verbose {
			fmt.Printf("trying relay: %s\n", k)
		}
		rl, e := relays.RelayConnect(ctx, k)
		if e != nil {
			if cfg.verbose {
				fmt.Fprintln(os.Stderr, e.Error())
			}
			continue
		}
		return rl
	}
	return nil
}

// Do is
func (cfg *Config) Do(r RelayPerms, f func(context.T, *relays.Relay) bool) {
	var wg sync.WaitGroup
	ctx := context.Bg()
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
		go func(wg *sync.WaitGroup, k string, v RelayPerms) {
			defer wg.Done()
			rl, e := relays.RelayConnect(ctx, k)
			if e != nil {
				if cfg.verbose {
					fmt.Fprintln(os.Stderr, e)
				}
				return
			}
			if !f(ctx, rl) {
				ctx.Done()
			}
			rl.Close()
		}(&wg, k, v)
	}
	wg.Wait()
}

func (cfg *Config) save(profile string) (e error) {
	if cfg.tempRelay {
		return nil
	}
	dir, e := configDir()
	if e != nil {
		return e
	}
	dir = filepath.Join(dir, "algia")

	var fp string
	if profile == "" {
		fp = filepath.Join(dir, "config.json")
	} else {
		fp = filepath.Join(dir, "config-"+profile+".json")
	}
	b, e := json.MarshalIndent(&cfg, "", "  ")
	if e != nil {
		return e
	}
	return ioutil.WriteFile(fp, b, 0644)
}

// Decode is
func (cfg *Config) Decode(ev *event.T) (e error) {
	var sk string
	var pub string
	if _, s, e := nip19.Decode(cfg.PrivateKey); e == nil {
		sk = s.(string)
		if pub, e = keys.GetPublicKey(s.(string)); e != nil {
			return e
		}
	} else {
		return e
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
	ss, e := nip04.ComputeSharedSecret(sp, sk)
	if e != nil {
		return e
	}
	content, e := nip04.Decrypt(ev.Content, ss)
	if e != nil {
		return e
	}
	ev.Content = content
	return nil
}

// PrintEvents is
func (cfg *Config) PrintEvents(evs []*event.T, followsMap map[string]Profile, j, extra bool) {
	if j {
		if extra {
			var events []Event
			for _, ev := range evs {
				if profile, ok := followsMap[ev.PubKey]; ok {
					events = append(events, Event{
						Event:   ev,
						Profile: profile,
					})
				}
			}
			for _, ev := range events {
				json.NewEncoder(os.Stdout).Encode(ev)
			}
		} else {
			for _, ev := range evs {
				json.NewEncoder(os.Stdout).Encode(ev)
			}
		}
		return
	}

	for _, ev := range evs {
		profile, ok := followsMap[ev.PubKey]
		if ok {
			color.Set(color.FgHiRed)
			fmt.Print(profile.Name)
		} else {
			color.Set(color.FgRed)
			fmt.Print(ev.PubKey)
		}
		color.Set(color.Reset)
		fmt.Print(": ")
		color.Set(color.FgHiBlue)
		fmt.Println(ev.PubKey)
		color.Set(color.Reset)
		fmt.Println(ev.Content)
	}
}

// Events is
func (cfg *Config) Events(f filter.T) []*event.T {
	var mu sync.Mutex
	found := false
	var m sync.Map
	cfg.Do(RelayPerms{Read: true}, func(ctx context.T, rl *relays.Relay) bool {
		mu.Lock()
		if found {
			mu.Unlock()
			return false
		}
		mu.Unlock()
		evs, e := rl.QuerySync(ctx, f)
		if e != nil {
			return true
		}
		for _, ev := range evs {
			if _, ok := m.Load(ev.ID); !ok {
				if ev.Kind == event.KindEncryptedDirectMessage {
					if e := cfg.Decode(ev); e != nil {
						continue
					}
				}
				m.LoadOrStore(ev.ID, ev)
				if len(f.IDs) == 1 {
					mu.Lock()
					found = true
					ctx.Done()
					mu.Unlock()
					break
				}
			}
		}
		return true
	})

	keys := []string{}
	m.Range(func(k, v any) bool {
		keys = append(keys, k.(string))
		return true
	})
	sort.Slice(keys, func(i, j int) bool {
		lhs, ok := m.Load(keys[i])
		if !ok {
			return false
		}
		rhs, ok := m.Load(keys[j])
		if !ok {
			return false
		}
		return lhs.(*event.T).CreatedAt.Time().Before(rhs.(*event.T).CreatedAt.Time())
	})
	var evs []*event.T
	for _, key := range keys {
		vv, ok := m.Load(key)
		if !ok {
			continue
		}
		evs = append(evs, vv.(*event.T))
	}
	return evs
}

func doVersion(cCtx *cli.Context) (e error) {
	fmt.Println(version)
	return nil
}

func main() {
	app := &cli.App{
		Usage:       "A cli application for nostr",
		Description: "A cli application for nostr",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "a", Usage: "profile name"},
			&cli.StringFlag{Name: "relays", Usage: "relays"},
			&cli.BoolFlag{Name: "V", Usage: "verbose"},
		},
		Commands: []*cli.Command{
			{
				Name:    "timeline",
				Aliases: []string{"tl"},
				Usage:   "show timeline",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "n", Value: 30, Usage: "number of items"},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
					&cli.BoolFlag{Name: "extra", Usage: "extra JSON"},
				},
				Action: doTimeline,
			},
			{
				Name:  "stream",
				Usage: "show stream",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "author"},
					&cli.IntSliceFlag{Name: "kind", Value: cli.NewIntSlice(event.KindTextNote)},
					&cli.BoolFlag{Name: "follow"},
					&cli.StringFlag{Name: "pattern"},
					&cli.StringFlag{Name: "reply"},
				},
				Action: doStream,
			},
			{
				Name:    "post",
				Aliases: []string{"n"},
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: "u", Usage: "users"},
					&cli.BoolFlag{Name: "stdin"},
					&cli.StringFlag{Name: "sensitive"},
					&cli.StringSliceFlag{Name: "emoji"},
					&cli.StringFlag{Name: "geohash"},
				},
				Usage:     "post new note",
				UsageText: "algia post [note text]",
				HelpName:  "post",
				ArgsUsage: "[note text]",
				Action:    doPost,
			},
			{
				Name:    "reply",
				Aliases: []string{"r"},
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "stdin"},
					&cli.StringFlag{Name: "id", Required: true},
					&cli.BoolFlag{Name: "quote"},
					&cli.StringFlag{Name: "sensitive"},
					&cli.StringSliceFlag{Name: "emoji"},
					&cli.StringFlag{Name: "geohash"},
				},
				Usage:     "reply to the note",
				UsageText: "algia reply --id [id] [note text]",
				HelpName:  "reply",
				ArgsUsage: "[note text]",
				Action:    doReply,
			},
			{
				Name:    "repost",
				Aliases: []string{"b"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
				Usage:     "repost the note",
				UsageText: "algia repost --id [id]",
				HelpName:  "repost",
				Action:    doRepost,
			},
			{
				Name:    "unrepost",
				Aliases: []string{"B"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
				Usage:     "unrepost the note",
				UsageText: "algia unrepost --id [id]",
				HelpName:  "unrepost",
				Action:    doUnrepost,
			},
			{
				Name:    "like",
				Aliases: []string{"l"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
					&cli.StringFlag{Name: "content"},
					&cli.StringFlag{Name: "emoji"},
				},
				Usage:     "like the note",
				UsageText: "algia like --id [id]",
				HelpName:  "like",
				Action:    doLike,
			},
			{
				Name:    "unlike",
				Aliases: []string{"L"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
				Usage:     "unlike the note",
				UsageText: "algia unlike --id [id]",
				HelpName:  "unlike",
				Action:    doUnlike,
			},
			{
				Name:    "delete",
				Aliases: []string{"d"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
				Usage:     "delete the note",
				UsageText: "algia delete --id [id]",
				HelpName:  "delete",
				Action:    doDelete,
			},
			{
				Name:    "search",
				Aliases: []string{"s"},
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "n", Value: 30, Usage: "number of items"},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
					&cli.BoolFlag{Name: "extra", Usage: "extra JSON"},
				},
				Usage:     "search notes",
				UsageText: "algia search [words]",
				HelpName:  "search",
				Action:    doSearch,
			},
			{
				Name:  "dm-list",
				Usage: "show DM list",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
				},
				Action: doDMList,
			},
			{
				Name:  "dm-timeline",
				Usage: "show DM timeline",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "u", Value: "", Usage: "DM user", Required: true},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
					&cli.BoolFlag{Name: "extra", Usage: "extra JSON"},
				},
				Action: doDMTimeline,
			},
			{
				Name: "dm-post",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "u", Value: "", Usage: "DM user", Required: true},
					&cli.BoolFlag{Name: "stdin"},
					&cli.StringFlag{Name: "sensitive"},
				},
				Usage:     "post new note",
				UsageText: "algia post [note text]",
				HelpName:  "post",
				ArgsUsage: "[note text]",
				Action:    doDMPost,
			},
			{
				Name: "profile",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "u", Value: "", Usage: "user"},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
				},
				Usage:     "show profile",
				UsageText: "algia profile",
				HelpName:  "profile",
				Action:    doProfile,
			},
			{
				Name:      "powa",
				Usage:     "post ぽわ〜",
				UsageText: "algia powa",
				HelpName:  "powa",
				Action:    doPowa,
			},
			{
				Name:      "puru",
				Usage:     "post ぷる",
				UsageText: "algia puru",
				HelpName:  "puru",
				Action:    doPuru,
			},
			{
				Name: "zap",
				Flags: []cli.Flag{
					&cli.Uint64Flag{Name: "amount", Usage: "amount for zap", Value: 1},
					&cli.StringFlag{Name: "comment", Usage: "comment for zap", Value: ""},
				},
				Usage:     "zap [note|npub|nevent]",
				UsageText: "algia zap [note|npub|nevent]",
				HelpName:  "zap",
				Action:    doZap,
			},
			{
				Name:      "version",
				Usage:     "show version",
				UsageText: "algia version",
				HelpName:  "version",
				Action:    doVersion,
			},
		},
		Before: func(cCtx *cli.Context) (e error) {
			if cCtx.Args().Get(0) == "version" {
				return nil
			}
			profile := cCtx.String("a")
			cfg, e := loadConfig(profile)
			if e != nil {
				return e
			}
			cCtx.App.Metadata = map[string]any{
				"config": cfg,
			}
			cfg.verbose = cCtx.Bool("V")
			relays := cCtx.String("relays")
			if strings.TrimSpace(relays) != "" {
				cfg.Relays = make(map[string]RelayPerms)
				for _, rl := range strings.Split(relays, ",") {
					cfg.Relays[rl] = RelayPerms{
						Read:  true,
						Write: true,
					}
				}
				cfg.tempRelay = true
			}
			return nil
		},
	}

	if e := app.Run(os.Args); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
