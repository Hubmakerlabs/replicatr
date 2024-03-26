package app

import (
	"encoding/json"
	"errors"
	"os"
)

type ExportCmd struct {
	ToFile string `arg:"-f,--tofile" help:"write to file instead of stdout"`
}

type ImportCmd struct {
	FromFile []string `arg:"-f,--fromfile,separate" help:"read from files instead of stdin (can use flag repeatedly for multiple files)"`
}

type InitCfg struct{}
type WipeBDB struct{}

type Config struct {
	ExportCmd    *ExportCmd `arg:"subcommand:export" json:"-" help:"export database as line structured JSON"`
	ImportCmd    *ImportCmd `arg:"subcommand:import" json:"-" help:"import data from line structured JSON"`
	InitCfgCmd   *InitCfg   `arg:"subcommand:initcfg" json:"-" help:"initialize relay configuration files"`
	Wipe         *WipeBDB   `arg:"subcommand:wipebdb" json:"-" help:"empties database"`
	Listen       string     `arg:"-l,--listen" default:"0.0.0.0:3334" json:"listen" help:"network address to listen on"`
	EventStore   string     `arg:"-e,--eventstore" default:"ic" json:"eventstore" help:"select event store backend [ic,badger]"`
	CanisterAddr string     `arg:"-C,--canisteraddr" default:"127.0.0.1:46847" json:"canister_addr" help:"IC canister address to use"`
	CanisterID   string     `arg:"-I,--canisterid" json:"canister_id" help:"IC canister ID to use"`
	Profile      string     `arg:"-p,--profile" json:"-" default:"replicatr" help:"profile name to use for storage"`
	Name         string     `arg:"-n,--name" json:"name" default:"replicatr relay" help:"name of relay for NIP-11"`
	Description  string     `arg:"-d,--description" json:"description" help:"description of relay for NIP-11"`
	Pubkey       string     `arg:"--pubkey" json:"pubkey" help:"public key of relay operator"`
	Contact      string     `arg:"-c,--contact" json:"contact,omitempty" help:"non-nostr relay operator contact details"`
	Icon         string     `arg:"-i,--icon" json:"icon" default:"https://i.nostr.build/n8vM.png" help:"icon to show on relay information pages"`
	AuthRequired bool       `arg:"-a,--auth" json:"auth_required" default:"false" help:"NIP-42 authentication required for all access"`
	Public       bool       `arg:"--public" json:"public" default:"true" help:"allow public read access to users not on ACL"`
	Owners       []string   `arg:"-o,--owner,separate" json:"owners" help:"specify public keys of users with owner level permissions on relay"`
	SecKey       string     `arg:"-s,--seckey" json:"seckey" help:"identity key of relay, used to sign 30066 and 30166 events and for message control interface"`
	// Whitelist permits ONLY inbound connections from specified IP addresses.
	Whitelist []string `arg:"-w,--whitelist,separate" json:"ip_whitelist" help:"IP addresses that are only allowed to access"`
	// AllowIPs is for bypassing authentication required for clients based on IP
	// addresses... primarily for testing with wireguard VPN clients run by the
	// developer, as these are stable, non-routeable addresses, this skips the
	// requirement enforced by AuthRequired.
	AllowIPs []string `arg:"-A,--allow,separate" json:"allow_ip" help:"IP addresses that are always allowed to access"`
	// DBSizeLimit configures a target maximum size to maintain the local
	// event store cache at, in megabytes (1,000,000 bytes).
	DBSizeLimit int `arg:"-S,--sizelimit" json:"db_size_limit" default:"0" help:"set the maximum size of the badger event store in megabytes"`
	// DBLowWater is the proportion of the DBSizeLimit to prune the database
	// down to when performing a garbage collection run.
	DBLowWater int `arg:"-L,--lowwater" json:"db_low_water" default:"75" help:"set target percentage for database size during garbage collection"`
	// DBHighWater is the proportion of the DBSizeLimit at which a garbage
	// collection run is triggered.
	DBHighWater int `arg:"-H,--highwater" json:"db_high_water" default:"90" help:"set garbage collection trigger percentage for database size during garbage collection"`
	// GCFrequency is the frequency to run a check on the database size and
	// if it breaches DBHighWater to prune it back to DBLowWater percentage
	// of DBSizeLimit in minutes.
	GCFrequency int    `arg:"-G,--gcfreq" json:"gc_frequency" default:"60" help:"frequency in seconds to check if database needs garbage collection"`
	MaxProcs    int    `arg:"-m" json:"max_procs" default:"128" help:"maximum number of goroutines to use"`
	LogLevel    string `arg:"--loglevel" default:"info" help:"set log level [off,fatal,error,warn,info,debug,trace] (can also use GODEBUG environment variable)"`
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
