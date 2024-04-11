package app

const Name = "digestr"

type Config struct {
	Seed         *int   `arg:"-s,--seed" json:"seed" help:"integer to use for random generation of events and queries"`
	EventAmount  int    `arg:"-e,--events" default:"1000" json:"EventAmonunt" help:"number of randomly generated events"`
	QueryAmount  int    `arg:"-q,--queries" default:"250" json:"QueryAmount" help:"number of randomly generated queries" `
	SkipSetup    bool   `arg:"--skipsetup" default:"false" json:"-" help:"execute before and after clause?"`
	CanisterAddr string `arg:"-c,--canisteraddr" default:"https://icp0.io/" json:"canister_addr" help:"IC canister address to use (for local, use 127.0.0.1:46847)"`
	CanisterID   string `arg:"-i,--canisterid" default:"rpfa6-ryaaa-aaaap-qccvq-cai" json:"canister_id" help:"IC canister ID to use"`
	Wipe         bool   `arg:"--wipe" default:"false" json:"-" help:"only wipe canister and badger"`
	LogLevel     string `arg:"--loglevel" default:"info" help:"set log level [off,fatal,error,warn,info,debug,trace] (can also use GODEBUG environment variable)"`
}
