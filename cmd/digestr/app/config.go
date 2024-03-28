package app

const Name = "digestr"

type SkipSetup struct{}

type Config struct {
	Seed         string     `arg:"-s,--seed" json:"seed" help:"integer to use for random generation of events and queries"`
	EventAmount  string     `arg:"-e,--events" default: "1000" json: "EventAmonunt" help:"number of randomly generated events"`
	QueryAmount  string     `arg: "-q,--queries" default: "250" json: "QueryAmount" help: "number of randomly generated queries" `
	SkipSetupCmd *SkipSetup `arg:"subcommand:skipsetup" json:"-" help:"skips execupting before and after clause"`
}
