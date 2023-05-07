package main

import (
	"fmt"
)

func main() {
	f, close, err := NewFirestore("./.keys/test-bfcae-firebase-adminsdk-jhjzx-65a328f380.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer close()

	mig := NewMigrator("./local", f, "test")
	a := map[string]any{
		"a": 123,
		"b": 456,
		"c": []int{7, 10, 9},
		"e": map[string]any{
			"d": 1,
			"f": true,
			"g": "hello",
		},
	}
	b := map[string]any{
		"a": 125,
		"c": []int{7, 8},
		"d": true,
		"e": map[string]any{
			"g": "goodbye",
		},
	}
	// sample workflow
	err = mig.Stage().Add("fig", a, "")
	if err != nil {
		fmt.Println(err.Error())
	}
	err = mig.Stage().Add("fig", b, "")
	if err != nil {
		fmt.Println(err.Error())
	}
	err = mig.PrepMigration()
	if err != nil {
		fmt.Println(err.Error())
	}
	mig.PresentMigration()
}
