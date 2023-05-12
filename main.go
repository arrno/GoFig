package main

import (
	"fmt"
	"time"
)

func main() {

	conf := Config{
		keyPath:     "./.keys/test-bfcae-firebase-adminsdk-jhjzx-65a328f380.json",
		storagePath: "./local",
		name:        "initial",
	}

	fig, err := NewController(conf)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer fig.Close()

	// set initial state
	d := map[string]any{
		"a": time.Now(),
		"b": time.Now(),
		"c": fig.RefField("fig/DLMwCPG41s2p_dcQrYA3"),
		"d": fig.RefField("fig/test"),
	}
	fig.Stage().Set("fig/fog", d, "")
	fig.ManageStagedMigration(false)

	// set updated state
	fig.mig.name = "updated"
	fig.mig.changes = []*Change{}
	dd := map[string]any{
		"a": time.Now(),
		"d": "fig/test",
		"c": fig.DeleteField(),
	}
	fig.Stage().Update("fig/fog", dd, "")
	fig.ManageStagedMigration(false)

	// rollback updated state
	fig.mig.name = "updated_rollback"
	fig.ManageStagedMigration(true)

}
