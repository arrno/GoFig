package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

// <----------------------------------------- Mock ------------------------------------------->

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

// <----------------------------------------- Global vars ------------------------------------------->

var before = map[string]any{
	"a": "foo",
	"b": "bar",
	"c": []any{1, 2, 3, 4},
	"d": false,
	"e": map[string]any{
		"f": "foo",
		"g": 7.8,
	},
}
var patch = map[string]any{
	"a": "far",
	"c": []any{1, 2, 6},
	"d": true,
	"e": map[string]any{
		"f": false,
	},
	"h": 1000,
}
var after = map[string]any{
	"a": "far",
	"b": "bar",
	"c": []any{1, 2, 6},
	"d": true,
	"e": map[string]any{
		"f": false,
		"g": 7.8,
	},
	"h": 1000,
}

type testChange struct {
	before   map[string]any
	patch    map[string]any
	command  Command
	after    map[string]any
	rollback map[string]any
}

// <----------------------------------------- Tests ------------------------------------------->

// TestSerialization calls Serialize/Deserialize function and verifies proper results.
func TestSerialization(t *testing.T) {
	// TODO
}

// TestChanges verifies we are properly solving for baseline 'before + patch + command' change scenarios.
// Changes represent the core logic of this app.
func TestChanges(t *testing.T) {

	payloads := map[string]testChange{
		"before_patch_add": {
			before:   map[string]any{},
			patch:    before,
			command:  MigratorAdd,
			after:    before,
			rollback: map[string]any{"a": nil, "b": nil, "c": nil, "d": nil, "e": nil},
		},
		"before_patch_update": {
			before:   before,
			patch:    patch,
			command:  MigratorUpdate,
			after:    after,
			rollback: map[string]any{"a": "foo", "c": []any{1, 2, 3, 4}, "d": false, "e": map[string]any{"f": "foo"}, "h": nil},
		},
		"before_patch_delete": {
			before:   before,
			command:  MigratorDelete,
			patch:    patch,
			after:    map[string]any{},
			rollback: map[string]any{"a": "foo", "b": "bar", "c": []any{1, 2, 3, 4}, "d": false, "e": map[string]any{"f": "foo", "g": 7.8}},
		},
		"before_patch_set": {
			before:   before,
			command:  MigratorSet,
			patch:    patch,
			after:    patch,
			rollback: map[string]any{"a": "foo", "b": "bar", "c": []any{1, 2, 3, 4}, "d": false, "e": map[string]any{"f": "foo", "g": 7.8}, "h": nil},
		},
	}

	// baseline scenarios
	for k, v := range payloads {

		c := NewChange("test/test", payloads[k].before, payloads[k].patch, payloads[k].command, mf)
		c.SolveChange()

		if c.command != v.command {
			t.Fatalf("Mismatched command on %s", k)
		}

		cafter, _ := json.Marshal(c.after)
		vafter, _ := json.Marshal(v.after)

		if string(cafter) != string(vafter) {
			t.Fatalf("Mismatched after on %s", k)
		}

		if !reflect.DeepEqual(c.patch, v.patch) {
			t.Fatalf("Mismatched patch on %s", k)
		}

		crollback, _ := json.Marshal(c.rollback)
		vrollback, _ := json.Marshal(v.rollback)

		if !reflect.DeepEqual(crollback, vrollback) {
			t.Fatalf("Mismatched rollback on %s", k)
		}

	}

}
