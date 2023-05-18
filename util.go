package fig

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"runtime"
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
		return "__delete__<delete>__delete__"
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
			return "__timestamp__" + data.(time.Time).Format("2006-01-02T15:04:05.000Z") + "__timestamp__"
		}

		_, ok = data.(*firestore.DocumentRef)
		if ok && data.(*firestore.DocumentRef) != nil {
			path := data.(*firestore.DocumentRef).Path
			path = strings.Split(path, "/(default)/documents/")[1]
			return "__docref__" + path + "__docref__"
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
		if strings.HasPrefix(data.(string), "__timestamp__") {
			time, _ := time.Parse("2006-01-02T15:04:05.000Z", strings.Replace(data.(string), "__timestamp__", "", -1))
			return time

		} else if strings.HasPrefix(data.(string), "__docref__") {
			path := strings.Replace(data.(string), "__docref__", "", -1)
			ref := f.refField(path)
			return ref

		} else if strings.HasPrefix(data.(string), "__delete__") {
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

func toMapAny(data any) map[string]any {

	newMap := map[string]any{}
	rtype := reflect.TypeOf(data)

	if rtype.Kind() == reflect.Map {
		val := reflect.ValueOf(data)

		for _, e := range val.MapKeys(){
			if k, ok := e.Interface().(string); ok {
				newMap[k] = val.MapIndex(e).Interface()
			}

		}
	}

	return newMap
}

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