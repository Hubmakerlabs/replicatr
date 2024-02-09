package main

import (
	"time"
)

// GetFollows is
func (cfg *C) GetFollows(profile string, update bool) (profiles Follows, err error) {
	var pub string
	if pub, _, err = getPubFromSec(cfg.SecretKey); chk.D(err) {
		return
	}
	log.D.Ln("pub", pub)
	// get followers
	if (cfg.LastUpdated(time.Hour) && !cfg.tempRelay) ||
		len(cfg.Follows) == 0 || update {

		cfg.Lock()
		cfg.Follows = make(Follows)
		cfg.FollowsRelays = make(FollowsRelays)
		cfg.Unlock()
		m := make(Checklist)
		cfg.Do(readPerms, cfg.GetRelaysAndTags(pub, &m))
		// cfg.Lock()
		// log.D.S(m)
		// cfg.Unlock()
		log.D.F("found %d followers", len(m))
		if len(m) > 0 {
			var follows []string
			for k := range m {
				follows = append(follows, k)
			}
			for i := 0; i < len(follows); i += 500 {
				// Calculate the end index based on the current index and slice
				// length
				end := i + 500
				if end > len(follows) {
					end = len(follows)
				}
				log.D.Ln("getting follows' profile data")
				// get follower's descriptions
				cfg.Do(readPerms, cfg.PopulateFollows(&follows, &i, &end))
			}
		}
		cfg.Touch()
		if err = cfg.save(profile); chk.D(err) {
			return nil, err
		}
	}
	return cfg.Follows, nil
}
