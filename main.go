package main

import (
	"fmt"
)

func main() {

	conf := map[string]string{
		"keyPath":     "./.keys/test-bfcae-firebase-adminsdk-jhjzx-65a328f380.json",
		"storagePath": "./local",
		"name":        "test",
	}

	fig, err := NewController(conf)

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer fig.Close()

	// fig.Stage().Add("fig", map[string]any{}, "")
	// fig.Stage().Update("fig/test", map[string]any{}, "")
	// fig.Stage().Delete("fig/d")

	fig.ManageStagedMigration(true)

}
