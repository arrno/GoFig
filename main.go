package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	f, close, err := NewFirestore("./.keys/test-bfcae-firebase-adminsdk-jhjzx-65a328f380.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer close()
	sample := map[string]any{
		"var": f.DeleteField(),
	}
	bsl, err := json.Marshal(sample)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(string(bsl))
}
