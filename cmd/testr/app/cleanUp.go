package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Hubmakerlabs/replicatr/pkg/ic/agent"
)

func CanisterCleanUp(id string, addr string, SecKey string) error {
	var b *agent.Backend
	var err error
	if b, err = agent.New(nil, id, addr, SecKey); chk.E(err) {
		return err
	}

	//clear all events from canister
	var result string
	if err = b.ClearEvents(); chk.E(err) {
		return err
	} else {
		fmt.Printf("from canister %v: \"%v\"\n", id, result)
	}

	if addr != "https://icp0.io/" {
		fmt.Println("local canister being used")
	}

	fmt.Print("\n")

	return nil

}

func BadgerCleanUp() error {
	//load canisterID and canisterAddress from config.json
	var dataDirBase string
	var err error
	if dataDirBase, err = os.UserHomeDir(); chk.E(err) {
		return err
	}
	dataDir := filepath.Join(dataDirBase, "testDB")
	if err = os.RemoveAll(dataDir); chk.E(err) {
		return err
	} else {
		fmt.Println("testDB Badger Database Deleted")
	}
	return nil
}
