package main

import (
	"encoding/json"
	"io/ioutil"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/nsf/jsondiff"
)

func prettydiff(b map[string]any, a map[string]any) (string, error) {

	bm, err := json.Marshal(b)
	if err != nil {
		return "", err
	}

	am, err := json.Marshal(a)
	if err != nil {
		return "", err
	}

	opt := jsondiff.Options{
		Added: jsondiff.Tag{
			Begin: "+ ",
		},
		Removed: jsondiff.Tag{
			Begin: "- ",
		},
		ChangedSeparator: " -> ",
		Indent:           "    ",
		SkipMatches:      true,
	}

	_, s := jsondiff.Compare(bm, am, &opt)
	return s, nil

}

func getDiffPatch(original []byte, target []byte) ([]byte, error) {
	return jsonpatch.CreateMergePatch(original, target)
}

func applyDiffPatch(original []byte, patch []byte) ([]byte, error) {
	return jsonpatch.MergePatch(original, patch)
}

func storeJson(data any, storagePath string, fileName string) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(storagePath+"/"+fileName+".json", js, 0644)
	return err
}

func loadJson[T any](fullPath string, target *T) error {
	content, err := ioutil.ReadFile(fullPath + ".json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, target)
	if err != nil {
		return err
	}
	return nil
}
