package main

import (
	"fmt"
	"time"
)

func main() {

	conf := Config{
		keyPath:     "./.keys/test-bfcae-firebase-adminsdk-jhjzx-65a328f380.json",
		storagePath: "./local",
		name:        "update",
	}

	fig, err := NewController(conf)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer fig.Close()

	d := map[string]any{
		"a": fig.DeleteField(),
		"d": time.Now(),
		"z": fig.RefField("fig/DLMwCPG41s2p_dcQrYA3"),
	}
	// fmt.Println(SerializeData(d, fig.mig.database))
	fig.Stage().Update("fig/CneTLRz-5nY8prwhdHJq", d, "")
	fig.ManageStagedMigration(false)

}
