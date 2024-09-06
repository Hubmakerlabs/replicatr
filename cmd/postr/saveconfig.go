package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func (cfg *C) save(profile string) (err error) {
	if cfg.tempRelay {
		return nil
	}
	if len(cfg.Relays) == 0 {
		log.D.Ln("not saving config with no relays, possibly was lost")
	}
	var dir string
	dir, err = configDir()
	if chk.D(err) {
		return err
	}
	dir = filepath.Join(dir, appName)

	var fp string
	if profile == "" {
		fp = filepath.Join(dir, "config.json")
	} else {
		fp = filepath.Join(dir, "config-"+profile+".json")
	}
	var b []byte
	b, err = json.MarshalIndent(&cfg, "", "\t")
	if chk.D(err) {
		return err
	}
	if len(cfg.Follows) == 0 {
		log.D.Ln("failed to get follow events")
		return
	}
	log.D.F("saving to file '%s'\n%s", fp, string(b))
	// return nil
	return os.WriteFile(fp, b, 0644)
}
