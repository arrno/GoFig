package main

import (
	"reflect"
	"testing"
)

type MockFirestore struct{}

func (f MockFirestore) GetDocData(docPath string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f MockFirestore) GenDocPath(colPath string) (string, error) {
	return "", nil
}
func (f MockFirestore) UpdateDoc(docPath string, data map[string]any) error {
	return nil
}
func (f MockFirestore) SetDoc(docPath string, data map[string]any) error {
	return nil
}
func (f MockFirestore) DeleteDoc(docPath string) error {
	return nil
}
func (f MockFirestore) DeleteField() any {
	return nil
}
func (f MockFirestore) RefField(docPath string) any {
	return nil
}
func (f MockFirestore) Name() string {
	return ""
}

var mf MockFirestore = MockFirestore{}

// TestSerialization calls Serialize/Deserialize function and verifies proper results.
func TestSerialization(t *testing.T) {
	// TODO
}

// TestChange verifies we are properly solving for all change scenarios.
// Changes represent the core logic of this app.
func TestChange(t *testing.T) {

	before := map[string]any{
		"a": "foo",
		"b": "bar",
		"c": []int{1, 2, 3, 4},
		"d": false,
		"e": map[string]any{
			"f": "foo",
			"g": 7.8,
		},
	}
	patch := map[string]any{
		"a": "far",
		"c": []int{1, 2, 6},
		"d": true,
		"e": map[string]any{
			"f": false,
		},
		"h": 1000,
	}
	after := map[string]any{
		"a": "far",
		"c": []int{1, 2, 6},
		"d": true,
		"e": map[string]any{
			"f": false,
			"g": 7.8,
		},
		"h": 1000,
	}
	type testChange struct {
		before      map[string]any
		patch       map[string]any
		instruction string
		command     Command
		after       map[string]any
		rollback    string
	}
	payloads := map[string]testChange{
		"before_patch_add": {
			before:      map[string]any{},
			patch:       before,
			instruction: "",
			command:     MigratorAdd,
			after:       before,
			rollback:    "{\"a\":\"null\",\"b\":\"null\",\"c\":null,\"d\":null,\"e\":null",
		},
		"before_patch_update": {
			before:      before,
			patch:       patch,
			instruction: "",
			command:     MigratorAdd,
			after:       after,
			rollback:    "{\"a\":\"foo\",\"c\":[1,2,3,4],\"d\":false,\"e\":{\"f\":\"foo\"},\"h\":null}",
		},
		"before_patch_delete": {
			before:      before,
			command:     MigratorDelete,
			patch:       patch,
			instruction: "",
			after:       map[string]any{},
			rollback:    "{\"a\":\"foo\",\"b\":\"bar\",\"c\":[1,2,3,4],\"d\":false,\"e\":{\"f\":\"foo\",\"g\":7.8}}",
		},
		"before_patch_set": {
			before:      before,
			command:     MigratorSet,
			patch:       patch,
			instruction: "",
			after:       patch,
			rollback:    "{\"a\":\"foo\",\"b\":\"bar\",\"c\":[1,2,3,4],\"d\":false,\"e\":{\"f\":\"foo\",\"g\":\"7.8\"},\"h\":null}",
		},
		"before_instruction": {
			before:      before,
			command:     MigratorUnknown,
			patch:       map[string]any{},
			instruction: "{\"a\":\"far\",\"c\":[1,2,6],\"d\":true,\"e\":{\"f\":\"false\"},\"h\":100}",
			after:       after,
			rollback:    "{\"a\":\"foo\",\"c\":[1,2,3,4],\"d\":false,\"e\":{\"f\":\"foo\"},\"h\":null}",
		},
	}
	for k, v := range payloads {
		c := NewChange("test/test", payloads[k].before, payloads[k].patch, payloads[k].command, payloads[k].instruction, mf)
		c.SolveChange()
		if c.command != v.command {
			t.Fatalf("Mismatched command on %s", k)
		}
		if !reflect.DeepEqual(c.after, v.after) {
			t.Fatalf("Mismatched after on %s", k)
		}
		if !reflect.DeepEqual(c.patch, v.patch) {
			t.Fatalf("Mismatched patch on %s", k)
		}
		if c.rollback != v.rollback {
			t.Fatalf("Mismatched rollback on %s", k)
		}

	}

}
