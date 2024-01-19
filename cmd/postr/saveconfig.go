package main

import (
	"encoding/json"
	"path/filepath"
)

func (cfg *C) save(profile string) (e error) {
	if cfg.tempRelay {
		return nil
	}
	if len(cfg.Relays) == 0 {
		log.D.Ln("not saving config with no relays, possibly was lost")
	}
	var dir string
	dir, e = configDir()
	if log.Fail(e) {
		return e
	}
	dir = filepath.Join(dir, appName)

	var fp string
	if profile == "" {
		fp = filepath.Join(dir, "config.json")
	} else {
		fp = filepath.Join(dir, "config-"+profile+".json")
	}
	var b []byte
	b, e = json.MarshalIndent(&cfg, "", "\t")
	if log.Fail(e) {
		return e
	}
	log.D.F("saving to file '%s'\n%s", fp, string(b))
	return nil
	// return os.WriteFile(fp, b, 0644)
}
