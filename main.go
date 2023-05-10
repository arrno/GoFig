package main

import (
	"fmt"
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

}
