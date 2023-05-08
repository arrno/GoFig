package main

import (
	"fmt"
)

func main() {

	conf := map[string]string{
		"keyPath":     "./.keys/test-bfcae-firebase-adminsdk-jhjzx-65a328f380.json",
		"storagePath": "./local",
		"name":        "update_rollback",
	}

	fig, err := NewController(conf)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer fig.Close()

	// fig.Stage().Add("fig", map[string]any{}, "")
	// fig.Stage().Update("fig/CneTLRz-5nY8prwhdHJq", map[string]any{
	// 	"a": 126,
	// 	"c": []int {
	// 	  8,
	// 	  10,
	// 	},
	// 	"e": map[string]any {
	// 	  "f": false,
	// 	  "g": "goodbye",
	// 	},
	// 	"f": map[string]any {
	// 		"nested": "nest",
	// 	},
	// }, "")
	// fig.Stage().Delete("fig/d")

	fig.ManageStagedMigration(true)

}
