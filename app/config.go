package app

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
	Pubkey       string     `arg:"-k,--pubkey" json:"pubkey" help:"public key of relay operator"`
	Contact      string     `arg:"-c,--contact" json:"contact" help:"non-nostr relay operator contact details"`
	Icon         string     `arg:"-i,--icon" json:"icon" default:"https://i.nostr.build/n8vM.png" help:"icon to show on relay information pages"`
	Whitelist    []string   `arg:"-w,--whitelist,separate" json:"ip_whitelist" help:"IP addresses that are allowed to access"`
	AuthRequired bool       `arg:"-a,--auth" json:"auth_required" help:"NIP-42 authentication required for all access"`
	Public       bool       `arg:"-p,--public" json:"public" help:"allow public read access to users not on ACL"`
	Owners       []string   `arg:"-o,--owner,separate" json:"owners" help:"specify public keys of users with owner level permissions on relay"`
}
