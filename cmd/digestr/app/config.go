package app

const Name = "digestr"

type Config struct {
	Seed        *int `arg:"-s,--seed" json:"seed" help:"integer to use for random generation of events and queries"`
	EventAmount int  `arg:"-e,--events" default:"1000" json:"EventAmonunt" help:"number of randomly generated events"`
	QueryAmount int  `arg:"-q,--queries" default:"250" json:"QueryAmount" help:"number of randomly generated queries" `
	SkipSetup   bool `arg:"--skipsetup" default:"false" json:"-" help:"execute before and after clause?"`
}
