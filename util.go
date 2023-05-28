package fig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"cloud.google.com/go/firestore"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/fatih/color"
	"github.com/nsf/jsondiff"
)

type colorTheme struct {
	green  func(a ...interface{}) string
	red    func(a ...interface{}) string
	yellow func(a ...interface{}) string
	blue   func(a ...interface{}) string
}

var clrthm colorTheme

// clrTheme is a function to build and return clrthm of type colorTheme which is a singleton.
func clrTheme() *colorTheme {
	if clrthm.green == nil {
		green := color.New(color.Bold, color.FgGreen).SprintFunc()
		yellow := color.New(color.Bold, color.FgYellow).SprintFunc()
		red := color.New(color.Bold, color.FgRed).SprintFunc()
		blue := color.New(color.Bold, color.FgBlue).SprintFunc()

		clr := colorTheme{
			green,
			red,
			yellow,
			blue,
		}

		clrthm = clr
	}

	return &clrthm
}

// PrettyDiff returns the pretty formatted difference between a before and after map.
func prettyDiff(b map[string]any, a map[string]any) (string, error) {
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
			Begin: clrTheme().green("+ "),
		},
		Removed: jsondiff.Tag{
			Begin: clrTheme().red("- "),
		},
		ChangedSeparator: clrTheme().yellow(" -> "),
		Indent:           "    ",
		SkipMatches:      true,
	}

	_, s := jsondiff.Compare(bm, am, &opt)

	return s, nil

}

// GetDiffPatch produces the json patch instructions needed to transform the original to the target.
func getDiffPatch(original []byte, target []byte) ([]byte, error) {
	return jsonpatch.CreateMergePatch(original, target)
}

// ApplyDiffPatch applies the json diff patch to the original and returns the result.
func applyDiffPatch(original []byte, patch []byte) ([]byte, error) {
	return jsonpatch.MergePatch(original, patch)
}

// SerializeData converts timestamps, docrefs, and other complex objects into marked strings.
func serializeData(data any, f figFirestore) any {
	if reflect.DeepEqual(data, f.deleteField()) {
		return "<delete>!delete<delete>"
	}

	v := reflect.ValueOf(data)

	switch k := v.Kind(); k {
	case reflect.Map:
		newData := map[string]any{}
		for k, v := range toMapAny(data) {
			newData[k] = serializeData(v, f)
		}
		return newData

	case reflect.Slice:
		newData := []any{}
		for _, d := range toSliceAny(data) {
			newData = append(newData, serializeData(d, f))
		}
		return newData

	default:
		_, ok := data.(time.Time)
		if ok {
			return "<time>" + data.(time.Time).Format("2006-01-02T15:04:05.000Z") + "<time>"
		}

		_, ok = data.(*firestore.DocumentRef)
		if ok && data.(*firestore.DocumentRef) != nil {
			path := data.(*firestore.DocumentRef).Path
			path = strings.Split(path, "/(default)/documents/")[1]
			return "<ref>" + path + "<ref>"
		}

	}

	return data
}

// DeSerializeData converts marked strings into timestamps, docrefs, and other complex objects.
func deSerializeData(data any, f figFirestore) any {
	switch k := reflect.ValueOf(data).Kind(); k {

	case reflect.Map:
		newData := map[string]any{}
		for k, v := range toMapAny(data) {
			newData[k] = deSerializeData(v, f)
		}
		return newData

	case reflect.Slice:
		newData := []any{}
		for _, d := range toSliceAny(data) {
			newData = append(newData, deSerializeData(d, f))
		}
		return newData

	case reflect.String:
		if strings.HasPrefix(data.(string), "<time>") {
			time, _ := time.Parse("2006-01-02T15:04:05.000Z", strings.Replace(data.(string), "<time>", "", -1))
			return time

		} else if strings.HasPrefix(data.(string), "<ref>") {
			path := strings.Replace(data.(string), "<ref>", "", -1)
			ref := f.refField(path)
			return ref

		} else if strings.HasPrefix(data.(string), "<delete>") {
			return f.deleteField()

		}
	}

	return data
}

// StoreJson saves data as json to disc.
func storeJson(data any, storagePath string, fileName string) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(storagePath+"/"+fileName+".json", js, 0644)
	return err
}

// LoadJson reads json from disc and hydrates the data into the provided target.
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

// LoadFig wraps loadJson. It first attempts to load the content from the database but fails back to local storage.
func loadFig[T any](db figFirestore, path string, target *T) error {
	if strings.HasPrefix(path, "[firestore]/") {
		suffix := strings.Replace(path, "[firestore]/", "", 1)
		return db.getDocStruct(target, suffix)
	}
	return loadJson(path, target)
}

// Transform returns new data where instances of before are replaced with after. If after is nil, key is dropped.
// This function is recursive for slices and maps but not for nested structs.
func transform(data any, before any, after any) any {
	switch k := reflect.ValueOf(data).Kind(); k {
	case reflect.Map:
		newData := map[any]any{}
		for k, v := range data.(map[any]any) {
			if reflect.DeepEqual(before, v) && reflect.DeepEqual(after, nil) {
				continue
			}
			newData[k] = transform(v, before, after)
		}
		return newData
	case reflect.Slice:
		newData := []any{}
		for _, d := range data.([]any) {
			newData = append(newData, transform(d, before, after))
		}
		return newData
	default:
		if reflect.DeepEqual(data, before) {
			return after
		}
	}
	return data
}

func maxNum[T int | float32 | float64](a T, b T) T {
	if a > b {
		return a
	}
	return b
}

var clearMap map[string]func() = map[string]func(){
	"linux": func() {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	},
	"windows": func() {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	},
}

// clearTerm clears the terminal.
func clearTerm() {
	if runtime.GOOS == "windows" {
		clearMap["windows"]()
		return
	}
	clearMap["linux"]()
}

// longestLine returns the length and text of the longest line given an input string.
func longestLine(s string) (int, string) {
	subString := s
	maxLen := 0
	for _, line := range strings.Split(s, "\n") {
		l := utf8.RuneCountInString(line)
		if maxLen < l {
			maxLen = l
			subString = line
		}
	}
	return maxLen, subString
}

// toMapAny converts any data into map[string]any.
func toMapAny(data any) map[string]any {
	newMap := map[string]any{}
	rtype := reflect.TypeOf(data)

	if rtype.Kind() == reflect.Map {
		val := reflect.ValueOf(data)

		for _, e := range val.MapKeys() {
			if k, ok := e.Interface().(string); ok {
				newMap[k] = val.MapIndex(e).Interface()
			}
		}
	}
	return newMap
}

// toSliceAny converts any data into []any.
func toSliceAny(data any) []any {
	newSlice := []any{}
	rtype := reflect.TypeOf(data)

	if rtype.Kind() == reflect.Slice {
		val := reflect.ValueOf(data)

		for i := 0; i < val.Len(); i++ {
			newSlice = append(newSlice, val.Index(i).Interface())
		}
	}
	return newSlice
}

// getNullMapPaths finds paths in nested data that end with a nil value.
func getNullMapPaths(data map[string]any, path []string, results *[][]string) {
	for k, v := range data {
		if v == nil {
			nullPath := append(path, k)
			npCopy := make([]string, len(nullPath))
			copy(npCopy, nullPath)
			*results = append(*results, npCopy)
			continue
		}
		val := reflect.ValueOf(v)
		if val.Type().Kind() == reflect.Map {
			nullPath := append(path, k)
			getNullMapPaths(toMapAny(v), nullPath, results)
		}
		if val.Type().Kind() == reflect.Slice {
			nullPath := append(path, k)
			getNullSlicePaths(toSliceAny(v), nullPath, results)
		}
	}
}

// getNullSlicePaths finds paths in nested data that end with a nil value.
func getNullSlicePaths(data []any, path []string, results *[][]string) {
	for i := range data {
		if data[i] == nil {
			nullPath := append(path, fmt.Sprintf("%d",i))
			npCopy := make([]string, len(nullPath))
			copy(npCopy, nullPath)
			*results = append(*results, npCopy)
			continue
		}
		kind := reflect.TypeOf(data[i]).Kind()
		if kind == reflect.Map {
			nullPath := append(path, fmt.Sprintf("%d",i))
			getNullMapPaths(toMapAny(data[i]), nullPath, results)
		}
		if kind == reflect.Slice {
			nullPath := append(path, fmt.Sprintf("%d",i))
			getNullSlicePaths(toSliceAny(data[i]), nullPath, results)
		}
	}
}

// replaceMapValues traverses a deep map and replaces the end value with a new value.
func replaceMapValues(data *map[string]any, path []string, newValue any) {
	level := data
	for i, key := range path {
		if i == len(path) - 1 {
			(*level)[key] = newValue
		} else {
			nextLevel := (*level)[key]
			val := reflect.ValueOf(nextLevel)
			if val.Type().Kind() == reflect.Slice {
				sl := toSliceAny(nextLevel)
				(*level)[key] = sl
				replaceSliceValues(&sl, path[i+1:], newValue)
				break
			}
			if val.Type().Kind() == reflect.Map {
				nl := toMapAny(nextLevel)
				(*level)[key] = nl
				level = &nl
			}
		}
	}
}

// replaceSliceValues traverses a deep slice and replaces the end value with a new value.
func replaceSliceValues(data *[]any, path []string, newValue any) {
	level := data
	for i, key := range path {
		index, err := strconv.ParseInt(key, 0, 64)
		if err != nil {
			break
		}
		if i == len(path) - 1 {
			(*level)[index] = newValue
		} else {
			nextLevel := (*level)[index]
			val := reflect.ValueOf(nextLevel)
			if val.Type().Kind() == reflect.Slice {
				sl := toSliceAny(nextLevel)
				(*level)[index] = sl
				level = &sl
			}
			if val.Type().Kind() == reflect.Map {
				nl := toMapAny(nextLevel)
				(*level)[index] = nl
				replaceMapValues(&nl, path[i+1:], newValue)
				break
			}
		}
	}
}