package main

import (
	"fmt"
	"time"
)

func main() {

	conf := map[string]string{
		"keyPath":     "./.keys/test-bfcae-firebase-adminsdk-jhjzx-65a328f380.json",
		"storagePath": "./local",
		"name":        "update_2_rollback",
	}

	fig, err := NewController(conf)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer fig.Close()

	d := map[string]any{
		"a": fig.mig.database.DeleteField(),
		"d": time.Now(),
		"z": fig.mig.database.RefField("fig/DLMwCPG41s2p_dcQrYA3"),
	}
	// fmt.Println(SerializeData(d, fig.mig.database))
	fig.Stage().Update("fig/CneTLRz-5nY8prwhdHJq",d,"")
	fig.ManageStagedMigration(false)

}
