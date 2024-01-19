package main

import (
	"time"
)

// GetFollows is
func (cfg *C) GetFollows(profile string) (profiles Follows, e error) {
	var pub string
	if pub, _, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	log.D.Ln("pub", pub)
	// get followers
	if (cfg.LastUpdated(3*time.Hour) && !cfg.tempRelay) ||
		len(cfg.Follows) == 0 {

		cfg.Lock()
		cfg.Follows = make(Follows)
		cfg.Unlock()
		m := make(Checklist)
		cfg.Do(readPerms, cfg.GetRelaysAndTags(pub, &m))
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
		if e = cfg.save(profile); log.Fail(e) {
			return nil, e
		}
	}
	return cfg.Follows, nil
}
