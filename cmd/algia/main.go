package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/urfave/cli/v2"
)

var log = log2.GetStd()

const name = "algia"

const version = "0.0.54"

var revision = "HEAD"

func configDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		dir, e := os.UserHomeDir()
		if log.Fail(e) {
			return "", e
		}
		return filepath.Join(dir, ".config"), nil
	default:
		return os.UserConfigDir()
	}
}

func loadConfig(profile string) (*C, error) {
	dir, e := configDir()
	if log.Fail(e) {
		return nil, e
	}
	dir = filepath.Join(dir, "algia")

	var fp string
	if profile == "" {
		fp = filepath.Join(dir, "config.json")
	} else if profile == "?" {
		names, e := filepath.Glob(filepath.Join(dir, "config-*.json"))
		if log.Fail(e) {
			return nil, e
		}
		for _, n := range names {
			n = filepath.Base(n)
			n = strings.TrimLeft(n[6:len(n)-5], "-")
			fmt.Println(n)
		}
		os.Exit(0)
	} else {
		fp = filepath.Join(dir, "config-"+profile+".json")
	}
	log.Fail(os.MkdirAll(filepath.Dir(fp), 0700))

	b, e := os.ReadFile(fp)
	if log.Fail(e) {
		return nil, e
	}
	var cfg C
	e = json.Unmarshal(b, &cfg)
	if log.Fail(e) {
		return nil, e
	}
	if len(cfg.Relays) == 0 {
		cfg.Relays = map[string]*RelayPerms{}
		cfg.Relays["wss://relay.nostr.band"] = &RelayPerms{
			Read:   true,
			Write:  true,
			Search: true,
		}
	}
	return &cfg, nil
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
			if log.Fail(e) {
				return e
			}
			cCtx.App.Metadata = map[string]any{
				"config": cfg,
			}
			cfg.verbose = cCtx.Bool("V")
			if cfg.verbose {
				log2.SetLogLevel(log2.Debug)
			}
			relays := cCtx.String("relays")
			if strings.TrimSpace(relays) != "" {
				cfg.Relays = make(map[string]*RelayPerms)
				for _, rl := range strings.Split(relays, ",") {
					cfg.Relays[rl] = &RelayPerms{
						Read:  true,
						Write: true,
					}
				}
				cfg.tempRelay = true
			}
			return nil
		},
	}

	if e := app.Run(os.Args); log.Fail(e) {
		os.Exit(1)
	}
}
