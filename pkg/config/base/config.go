package base

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type ExportCmd struct {
	ToFile string `arg:"-f,--tofile" help:"write to file instead of stdout"`
}

type ImportCmd struct {
	FromFile     []string `arg:"-f,--fromfile,separate" help:"read from files instead of stdin (can use flag repeatedly for multiple files)"`
	StartingFrom int      `arg:"--importfrom" help:"start scanning import file from this position in bytes"`
}

type InitCfg struct{}
type WipeBDB struct{}
type RescanAC struct{}
type PubKey struct{}
type AddRelay struct {
	PubKey string `arg:"--addpubkey" help:"public key of client to add"`
	Admin  bool   `arg:"--admin"  help:"set client as admin"`
}
type RemoveRelay struct {
	PubKey string `arg:"--removepubkey" help:"public key of client to remove"`
}

type GetPermission struct {
}

func GetDefaultConfig() *Config {
	return &Config{
		Listen:        "0.0.0.0:3334",
		EventStore:    "ic",
		CanisterAddr:  "https://icp0.io/",
		Profile:       "replicatr",
		Name:          "replicatr relay",
		Icon:          "https://i.nostr.build/n8vM.png",
		AuthRequired:  false,
		Public:        true,
		DBLowWater:    86,
		DBHighWater:   92,
		GCFrequency:   300,
		MaxProcs:      4,
		LogLevel:      "info",
		GCRatio:       100,
		MemLimit:      500000000,
		PollFrequency: 5 * time.Second,
		PollOverlap:   4,
	}
}

type Config struct {
	InitCfgCmd       *InitCfg       `arg:"subcommand:initcfg" json:"-" help:"initialize relay configuration files"`
	ExportCmd        *ExportCmd     `arg:"subcommand:export" json:"-" help:"export database as line structured JSON"`
	ImportCmd        *ImportCmd     `arg:"subcommand:import" json:"-" help:"import data from line structured JSON"`
	PubKeyCmd        *PubKey        `arg:"subcommand:pubkey" json:"-" help:"print relay canister public key"`
	AddRelayCmd      *AddRelay      `arg:"subcommand:addrelay" json:"-" help:"add a relay to the cluster"`
	RemoveRelayCmd   *RemoveRelay   `arg:"subcommand:removerelay" json:"-" help:"remove a relay from the cluster"`
	GetPermissionCmd *GetPermission `arg:"subcommand:getpermission" json:"-" help:"get permission of a relay"`
	Wipe             *WipeBDB       `arg:"subcommand:wipebdb" json:"-" help:"empties local badger database (bdb)"`
	// Rescan           *RescanAC      `arg:"subcommand:rescan" json:"-" help:"clear and regenerate access counter records"`
	Listen       string   `arg:"-l,--listen" json:"listen" help:"network address to listen on"`
	EventStore   string   `arg:"-e,--eventstore" json:"eventstore" help:"select event store backend [ic,badger,iconly]"`
	CanisterAddr string   `arg:"-C,--canisteraddr" json:"canister_addr" help:"IC canister address to use (for local, use http://127.0.0.1:<port number>)"`
	CanisterId   string   `arg:"-I,--canisterid" json:"canister_id" help:"IC canister ID to use"`
	Profile      string   `arg:"-p,--profile" default:"replicatr" json:"-"  help:"profile name to use for storage"` // default:"replicatr"
	Name         string   `arg:"-n,--name" json:"name"  help:"name of relay for NIP-11"`                            // default:"replicatr relay"
	Description  string   `arg:"-d,--description" json:"description" help:"description of relay for NIP-11"`
	Pubkey       string   `arg:"--pubkey" json:"pubkey" help:"public key of relay operator"`
	Contact      string   `arg:"-c,--contact" json:"contact,omitempty" help:"non-nostr relay operator contact details"`
	Icon         string   `arg:"-i,--icon" json:"icon"  help:"icon to show on relay information pages"`                // default:"https://i.nostr.build/n8vM.png"
	AuthRequired bool     `arg:"-a,--auth" json:"auth_required"  help:"NIP-42 authentication required for all access"` // default:"false"
	Public       bool     `arg:"--public" json:"public"  help:"allow public read access to users not on ACL"`          // default:"true"
	Owners       []string `arg:"-o,--owner,separate" json:"owners" help:"specify public keys of users with owner level permissions on relay"`
	SecKey       string   `arg:"-s,--seckey" json:"seckey" help:"identity key of relay, used to sign 30066 and 30166 events and for message control interface"`
	// Whitelist permits ONLY inbound connections from specified IP addresses.
	Whitelist []string `arg:"-w,--whitelist,separate" json:"ip_whitelist" help:"IP addresses that are only allowed to access"`
	// AllowIPs is for bypassing authentication required for clients based on IP
	// addresses... primarily for testing with wireguard VPN clients run by the
	// developer, as these are stable, non-routeable addresses, this skips the
	// requirement enforced by AuthRequired.
	AllowIPs []string `arg:"-A,--allow,separate" json:"allow_ip" help:"IP addresses that are always allowed to access"`
	// DBSizeLimit configures a target maximum size to maintain the local
	// event store cache at, in megabytes (1,000,000 bytes).
	DBSizeLimit int `arg:"-S,--sizelimit" json:"db_size_limit" help:"set the maximum size of the badger event store in bytes"` // default:"0"
	// DBLowWater is the proportion of the DBSizeLimit to prune the database
	// down to when performing a garbage collection run.
	DBLowWater int `arg:"-L,--lowwater" json:"db_low_water" help:"set target percentage for database size during garbage collection"` // default:"86"
	// DBHighWater is the proportion of the DBSizeLimit at which a garbage
	// collection run is triggered.
	DBHighWater int `arg:"-H,--highwater" json:"db_high_water" help:"set garbage collection trigger percentage for database size during garbage collection"` // default:"92"
	// GCFrequency is the frequency to run a check on the database size and
	// if it breaches DBHighWater to prune it back to DBLowWater percentage
	// of DBSizeLimit in minutes.
	GCFrequency int    `arg:"-G,--gcfreq" json:"gc_frequency" help:"frequency in seconds to check if database needs garbage collection"`            // default:"300"
	MaxProcs    int    `arg:"--maxprocs" json:"max_procs" help:"maximum number of goroutines to use"`                                               // default:"128"
	LogLevel    string `arg:"--loglevel"  help:"set log level [off,fatal,error,warn,info,debug,trace] (can also use GODEBUG environment variable)"` // default:"info"
	PProf       bool   `arg:"--pprof" help:"enable CPU and memory profiling"`
	GCRatio     int    `arg:"--gcratio" help:"set GC percentage for triggering GC sweeps"`             // default:"100"
	MemLimit    int64  `arg:"--memlimit" help:"set memory limit on process to constrain memory usage"` // default:"500000000"
	// PollFrequency is how often the L2 is queried for recent events
	PollFrequency time.Duration `arg:"--pollfrequency" help:"if a level 2 event store is enabled how often it polls"`
	// PollOverlap is the multiple of the PollFrequency within which polling the L2
	// is done to ensure any slow synchrony on the L2 is covered (2-4 usually)
	PollOverlap timestamp.T `arg:"--polloverlap" help:"if a level 2 event store is enabled, multiple of poll freq overlap to account for latency"`
}

func (c *Config) Save(filename string) (err error) {
	if c == nil {
		err = errors.New("cannot save nil relay config")
		log.E.Ln(err)
		return
	}
	var b []byte
	if b, err = json.MarshalIndent(c, "", "    "); chk.E(err) {
		return
	}
	if err = os.WriteFile(filename, b, 0600); chk.E(err) {
		return
	}
	return
}

func (c *Config) Load(filename string) (err error) {
	if c == nil {
		err = errors.New("cannot load into nil config")
		chk.E(err)
		return
	}
	var b []byte
	if b, err = os.ReadFile(filename); chk.E(err) {
		return
	}
	// log.D.F("configuration\n%s", string(b))
	if err = json.Unmarshal(b, c); chk.E(err) {
		return
	}
	return
}
