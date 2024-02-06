package app

type ExportCmd struct {
	ToFile string `arg:"-f,--tofile" help:"write to file instead of stdout"`
}

type ImportCmd struct {
	FromFile []string `arg:"-f,--fromfile,separate" help:"read from files instead of stdin (can use flag repeatedly for multiple files)"`
}

type InitACL struct {
	Owner  string `arg:"positional,required" help:"initialize ACL configuration with an owner public key"`
	Public bool   `arg:"-p,--public" default:"false" help:"allow public read access"`
	Auth   bool   `arg:"-a,--auth" default:"false" help:"require auth for public access (recommended)"`
}

type InitCfg struct {
}

type Config struct {
	ExportCmd   *ExportCmd `json:"-" arg:"subcommand:export" help:"export database as line structured JSON"`
	ImportCmd   *ImportCmd `json:"-" arg:"subcommand:import" help:"import data from line structured JSON"`
	InitACLCmd  *InitACL   `json:"-" arg:"subcommand:initacl" help:"initialize access control configuration"`
	InitCfgCmd  *InitCfg   `json:"-" arg:"subcommand:initcfg" help:"initialize relay configuration files"`
	Listen      string     `json:"listen" arg:"-l,--listen" default:"0.0.0.0:3334" help:"network address to listen on"`
	Profile     string     `json:"-" arg:"-p,--profile" default:"replicatr" help:"profile name to use for storage"`
	Name        string     `json:"name" arg:"-n,--name" default:"replicatr relay" help:"name of relay for NIP-11"`
	Description string     `json:"description" arg:"--description" help:"description of relay for NIP-11"`
	Pubkey      string     `json:"pubkey" arg:"-k,--pubkey" help:"public key of relay operator"`
	Contact     string     `json:"contact" arg:"-c,--contact" help:"non-nostr relay operator contact details"`
	Icon        string     `json:"icon" arg:"-i,--icon" default:"https://i.nostr.build/n8vM.png" help:"icon to show on relay information pages"`
	Whitelist   []string   `arg:"-w,--whitelist,separate" help:"IP addresses that are allowed to access"`
}
