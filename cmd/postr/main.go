package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/slog"
	"github.com/urfave/cli/v2"
)

var log = slog.GetStd()

const appName = "postr"

const version = "0.0.54"

var revision = "HEAD"

func configDir() (dir string, err error) {
	switch runtime.GOOS {
	case "darwin":
		if dir, err = os.UserHomeDir(); log.Fail(err) {
			return
		}
		return filepath.Join(dir, ".config"), nil
	default:
		return os.UserConfigDir()
	}
}

func loadConfig(profile string) (cfg *C, err error) {
	var dir string
	if dir, err = configDir(); log.Fail(err) {
		return nil, err
	}
	dir = filepath.Join(dir, appName)
	var fp string
	switch profile {
	case "":
		fp = filepath.Join(dir, "config.json")
	case "?":
		var nn []string
		p := filepath.Join(dir, "config-*.json")
		if nn, err = filepath.Glob(p); log.Fail(err) {
			return
		}
		for _, n := range nn {
			n = filepath.Base(n)
			n = strings.TrimLeft(n[6:len(n)-5], "-")
			fmt.Println(n)
		}
		os.Exit(0)
	default:
		fp = filepath.Join(dir, "config-"+profile+".json")
	}
	if err = os.MkdirAll(filepath.Dir(fp), 0700); log.Fail(err) {
		return
	}
	var b []byte
	if b, err = os.ReadFile(fp); log.Fail(err) {
		return
	}
	cfg = new(C)
	if err = json.Unmarshal(b, cfg); log.Fail(err) {
		return
	}
	log.D.Ln("relays", cfg.Relays)
	if len(cfg.Relays) == 0 {
		cfg.Relays = Relays{
			"wss://relay.nostr.band": {
				Read:   true,
				Write:  true,
				Search: true,
			},
		}
	}
	return
}

func doVersion(_ *cli.Context) (err error) {
	fmt.Println(version)
	return nil
}

func main() {
	app := &cli.App{
		Usage:       "A cli application for nostr",
		Description: "A cli application for nostr",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "a", Usage: "profile appName"},
			&cli.StringFlag{Name: "relays", Usage: "relays"},
			&cli.BoolFlag{Name: "v", Usage: "verbose"},
			&cli.BoolFlag{Name: "t", Usage: "trace"},
		},
		Commands: []*cli.Command{
			{
				Name:    "timeline",
				Aliases: []string{"tl"},
				Usage:   "show timeline",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "n", Value: 30,
						Usage: "number of items"},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
					// &cli.BoolFlag{Name: "extra", Usage: "extra JSON"},
				},
				Action: Timeline,
			},
			{
				Name:  "get",
				Usage: "retrieves one or more based on a provided event ID hashes, space separated",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
				},
				UsageText: appName + " get [note ID]",
				HelpName:  "post",
				ArgsUsage: "[note ID]",
				Action:    Get,
			},
			// {
			// 	Name:  "stream",
			// 	Usage: "show stream",
			// 	Flags: []cli.Flag{
			// 		&cli.StringFlag{Name: "author"},
			// 		&cli.IntSliceFlag{Name: "kind", Value: cli.NewIntSlice(kind.TextNote)},
			// 		&cli.BoolFlag{Name: "follow"},
			// 		&cli.StringFlag{Name: "pattern"},
			// 		&cli.StringFlag{Name: "reply"},
			// 	},
			// 	Action: doStream,
			// },
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
				UsageText: appName + " post [note text]",
				HelpName:  "post",
				ArgsUsage: "[note text]",
				Action:    Post,
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
				UsageText: appName + " reply --id [id] [note text]",
				HelpName:  "reply",
				ArgsUsage: "[note text]",
				Action:    Reply,
			},
			{
				Name:    "repost",
				Aliases: []string{"b"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
				Usage:     "repost the note",
				UsageText: appName + " repost --id [id]",
				HelpName:  "repost",
				Action:    Repost,
			},
			// {
			// 	Name:    "unrepost",
			// 	Aliases: []string{"B"},
			// 	Flags: []cli.Flag{
			// 		&cli.StringFlag{Name: "id", Required: true},
			// 	},
			// 	Usage:     "unrepost the note",
			// 	UsageText: appName + " unrepost --id [id]",
			// 	HelpName:  "unrepost",
			// 	Action:    doUnrepost,
			// },
			{
				Name:    "like",
				Aliases: []string{"l"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
					&cli.StringFlag{Name: "content"},
					&cli.StringFlag{Name: "emoji"},
				},
				Usage:     "like the note",
				UsageText: appName + " like --id [id]",
				HelpName:  "like",
				Action:    Like,
			},
			// {
			// 	Name:    "unlike",
			// 	Aliases: []string{"L"},
			// 	Flags: []cli.Flag{
			// 		&cli.StringFlag{Name: "id", Required: true},
			// 	},
			// 	Usage:     "unlike the note",
			// 	UsageText: appName + " unlike --id [id]",
			// 	HelpName:  "unlike",
			// 	Action:    doUnlike,
			// },
			// {
			// 	Name:    "delete",
			// 	Aliases: []string{"d"},
			// 	Flags: []cli.Flag{
			// 		&cli.StringFlag{Name: "id", Required: true},
			// 	},
			// 	Usage:     "delete the note",
			// 	UsageText: appName + " delete --id [id]",
			// 	HelpName:  "delete",
			// 	Action:    doDelete,
			// },
			{
				Name:    "search",
				Aliases: []string{"s"},
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "n", Value: 30,
						Usage: "number of items"},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
					&cli.BoolFlag{Name: "extra", Usage: "extra JSON"},
				},
				Usage:     "search notes",
				UsageText: appName + " search [words]",
				HelpName:  "search",
				Action:    Search,
			},
			// {
			// 	Name:  "dm-list",
			// 	Usage: "show DM list",
			// 	Flags: []cli.Flag{
			// 		&cli.BoolFlag{Name: "json", Usage: "output JSON"},
			// 	},
			// 	Action: doDMList,
			// },
			// {
			// 	Name:  "dm-timeline",
			// 	Usage: "show DM timeline",
			// 	Flags: []cli.Flag{
			// 		&cli.StringFlag{Name: "u", Value: "", Usage: "DM user", Required: true},
			// 		&cli.BoolFlag{Name: "json", Usage: "output JSON"},
			// 		&cli.BoolFlag{Name: "extra", Usage: "extra JSON"},
			// 	},
			// 	Action: doDMTimeline,
			// },
			// {
			// 	Name: "dm-post",
			// 	Flags: []cli.Flag{
			// 		&cli.StringFlag{Name: "u", Value: "", Usage: "DM user", Required: true},
			// 		&cli.BoolFlag{Name: "stdin"},
			// 		&cli.StringFlag{Name: "sensitive"},
			// 	},
			// 	Usage:     "post new note",
			// 	UsageText: appName + " post [note text]",
			// 	HelpName:  "post",
			// 	ArgsUsage: "[note text]",
			// 	Action:    doDMPost,
			// },
			{
				Name: "profile",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "u", Value: "", Usage: "user"},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
				},
				Usage:     "show profile",
				UsageText: appName + " profile",
				HelpName:  "profile",
				Action:    Profile,
			},
			// {
			// 	Name: "zap",
			// 	Flags: []cli.Flag{
			// 		&cli.Uint64Flag{Name: "amount", Usage: "amount for zap", Value: 1},
			// 		&cli.StringFlag{Name: "comment", Usage: "comment for zap", Value: ""},
			// 	},
			// 	Usage:     "zap [note|npub|nevent]",
			// 	UsageText: appName + " zap [note|npub|nevent]",
			// 	HelpName:  "zap",
			// 	Action:    doZap,
			// },
			{
				Name:      "version",
				Usage:     "show version",
				UsageText: appName + " version",
				HelpName:  "version",
				Action:    doVersion,
			},
		},
		Before: func(cCtx *cli.Context) (err error) {
			if cCtx.Args().Get(0) == "version" {
				return nil
			}
			profile := cCtx.String("a")
			var cfg *C
			if cfg, err = loadConfig(profile); log.Fail(err) {
				return err
			}
			cCtx.App.Metadata = map[string]any{
				"config": cfg,
			}
			if cfg.verbose = cCtx.Bool("v"); cfg.verbose {
				slog.SetLogLevel(slog.Debug)
			}
			if cfg.trace = cCtx.Bool("t"); cfg.trace {
				slog.SetLogLevel(slog.Trace)
			}
			relays := cCtx.String("relays")
			if strings.TrimSpace(relays) != "" {
				cfg.Relays = make(Relays)
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
	if err := app.Run(os.Args); log.E.Chk(err) {
		os.Exit(1)
	}
}
