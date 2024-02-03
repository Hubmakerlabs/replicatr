package replicatr

import "time"

type Spammer struct {
	Address     string
	Offenses    int
	BannedUntil time.Time
}

type Spam struct {
	Spammers []Spam
}
