package main

import (
	"encoding/json"
	"io/ioutil"
	"reflect"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/nsf/jsondiff"
)

// PrettyDiff returns the pretty formatted difference between a before and after map.
func PrettyDiff(b map[string]any, a map[string]any) (string, error) {

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

// GetDiffPatch produces the json patch instructions needed to transform the original to the target.
func GetDiffPatch(original []byte, target []byte) ([]byte, error) {
	return jsonpatch.CreateMergePatch(original, target)
}

// ApplyDiffPatch applies the json diff patch to the original and returns the result.
func ApplyDiffPatch(original []byte, patch []byte) ([]byte, error) {
	return jsonpatch.MergePatch(original, patch)
}

// StoreJson saves data as json to disc.
func StoreJson(data any, storagePath string, fileName string) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(storagePath+"/"+fileName+".json", js, 0644)
	return err
}

// LoadJson reads json from disc and hydrates the data into the provided target.
func LoadJson[T any](fullPath string, target *T) error {
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

// Transform returns new data where instances of before are replaced with after. If after is nil, key is dropped.
// This function is recursive for slices and maps but not for nested structs.
func Transform(data any, before any, after any) any {
	switch k := reflect.ValueOf(data).Kind(); k {
	case reflect.Map:
		newData := map[any]any{}
		for k, v := range data.(map[any]any) {
			if reflect.DeepEqual(before, v) && reflect.DeepEqual(after, nil) {
				continue
			}
			newData[k] = Transform(v, before, after)
		}
		return newData
	case reflect.Slice:
		newData := []any{}
		for _, d := range data.([]any) {
			newData = append(newData, Transform(d, before, after))
		}
		return newData
	default:
		if reflect.DeepEqual(data, before) {
			return after
		}
	}
	return data
}
