package main

import (
	"encoding/json"
	"fmt"
)

func main() {

	// a := map[string]any{"a": 123, "b": 456, "c": []int{7, 10, 9}}
	// b := map[string]any{"a": 125, "c": []int{7, 8}, "d": true}

	// pdiff := prettydiff(a, b)
	// fmt.Println(pdiff)

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
	println("!!")
	am, err := json.Marshal(a)
	if err != nil {
		fmt.Println(err.Error())
	}
	bm, err := json.Marshal(b)
	if err != nil {
		fmt.Println(err.Error())
	}
	by, err := applyDiffPatch(am, bm)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(by))

	by, err = getDiffPatch(am, by)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(by))

}
