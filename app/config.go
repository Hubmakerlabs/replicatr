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

type InitCfg struct {
}

type Config struct {
	ExportCmd    *ExportCmd `arg:"subcommand:export" json:"-" help:"export database as line structured JSON"`
	ImportCmd    *ImportCmd `arg:"subcommand:import" json:"-" help:"import data from line structured JSON"`
	InitCfgCmd   *InitCfg   `arg:"subcommand:initcfg" json:"-" help:"initialize relay configuration files"`
	Listen       string     `arg:"-l,--listen" default:"0.0.0.0:3334" json:"listen" help:"network address to listen on"`
	Profile      string     `arg:"-p,--profile" json:"-" default:"replicatr" help:"profile name to use for storage"`
	Name         string     `arg:"-n,--name" json:"name" default:"replicatr relay" help:"name of relay for NIP-11"`
	Description  string     `arg:"-d,--description" json:"description" help:"description of relay for NIP-11"`
	Pubkey       string     `arg:"--pubkey" json:"pubkey" help:"public key of relay operator"`
	Contact      string     `arg:"-c,--contact" json:"contact,omitempty" help:"non-nostr relay operator contact details"`
	Icon         string     `arg:"-i,--icon" json:"icon" default:"https://i.nostr.build/n8vM.png" help:"icon to show on relay information pages"`
	Whitelist    []string   `arg:"-w,--whitelist,separate" json:"ip_whitelist" help:"IP addresses that are allowed to access"`
	AuthRequired bool       `arg:"-a,--auth" json:"auth_required" default:"false" help:"NIP-42 authentication required for all access"`
	Public       bool       `arg:"--public" json:"public" default:"true" help:"allow public read access to users not on ACL"`
	Owners       []string   `arg:"-o,--owner,separate" json:"owners" help:"specify public keys of users with owner level permissions on relay"`
	SecKey       string     `arg:"-s" json:"seckey" help:"identity key of relay, used to sign 30066 and 30166 events and for message control interface"`
	MaxProcs     int        `args:"-m" json:"max_procs" default:"128" help:"maximum number of goroutines to use"`
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
	log.D.F("configuration\n%s", string(b))
	if err = json.Unmarshal(b, c); chk.E(err) {
		return
	}
	return
}
