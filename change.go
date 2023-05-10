package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// <---------------------- Change ------------------------------------>

// Command is an enum of supported Change command types for example 'Update'.
type Command int

const (
	MigratorUnknown Command = iota
	MigratorUpdate
	MigratorSet
	MigratorAdd
	MigratorDelete
)

// Change represents one change on one document. A change must contain enough data points to be solved.
// For example, given a before and a patch, we can solve for the after value.
type Change struct {
	docPath     string
	before      map[string]any
	patch       map[string]any
	after       map[string]any
	instruction string
	command     Command
	prettyDiff  string
	rollback    string
	errState    error
	database    Firestore
	cache       map[string]map[string]any
}

// NewChange is a Change factory.
func NewChange(docPath string,
	before map[string]any,
	patch map[string]any,
	command Command,
	instruction string,
	database Firestore) *Change {
	c := Change{
		docPath:     docPath,
		before:      before,
		patch:       patch,
		command:     command,
		instruction: instruction,
		database:    database,
		errState:    errors.New("Change has not yet been solved."),
	}
	return &c
}

// SolveChange will attempt to solve for all needed Change values given the current state.
func (c *Change) SolveChange() error {
	c.errState = nil
	err := c.inferAfter()
	if err != nil {
		c.errState = err
		return err
	}
	// if this is a rollback...
	err = c.inferCommand()
	if err != nil {
		c.errState = err
		return err
	}
	err = c.inferPrettyDiff()
	if err != nil {
		c.errState = err
		return err
	}
	err = c.inferRollback()
	if err != nil {
		c.errState = err
		return err
	}
	c.inferPatch()
	return nil
}

// commandString converts a command enum to a string.
func (c *Change) commandString() string {
	switch c.command {
	case MigratorUpdate:
		return "update"
	case MigratorSet:
		return "set"
	case MigratorAdd:
		return "add"
	case MigratorDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// inferAfter attempts to solve for the Change's after value.
func (c *Change) inferAfter() error {

	needsDeserialize := false

	if c.command != MigratorUnknown {
		switch c.command {
		case MigratorSet:
			c.after = c.patch
			return nil
		case MigratorAdd:
			c.after = c.patch
			return nil
		case MigratorDelete:
			c.after = map[string]any{}
			return nil
		}

	}
	if c.before == nil || (c.patch == nil && c.instruction == "") {
		return errors.New("Need before and patch/instruction to infer after.")

	}
	bm, err := json.Marshal(c.before)
	if err != nil {
		return err
	}

	var pm []byte
	if c.patch != nil {
		pm, err = json.Marshal(c.patch)
	} else {
		needsDeserialize = true
		pm = []byte(c.instruction)
	}
	if err != nil {
		return err
	}
	after, err := ApplyDiffPatch(bm, pm)
	if err != nil {
		return err
	}

	var ua map[string]any
	json.Unmarshal(after, &ua)
	c.after = ua
	if needsDeserialize {
		c.after = DeSerializeData(c.after, c.database).(map[string]any)
	}
	return nil

}

// inferCommand attempts to solve for the Change's command value.
func (c *Change) inferCommand() error {
	// this is only really needed for rollbacks
	if c.command != MigratorUnknown {
		return nil
	}

	if c.after == nil {
		return errors.New("Need after value to infer command.")
	}

	// {}->{...}/{...}->{...} are set... {...}->{} is delete
	if len(c.after) > 0 {
		c.command = MigratorSet
	} else {
		c.command = MigratorDelete
	}

	return nil
}

// inferRollback attempts to solve for the Change's rollback value.
// The rollback value is in the form of json patch instructions.
func (c *Change) inferRollback() error {
	if c.before == nil || c.after == nil {
		return errors.New("Need before and after value to infer rollback.")
	}
	sBefore, sAfter := c.beforeAfterCache()
	a, err := json.Marshal(sAfter)
	if err != nil {
		return err
	}
	b, err := json.Marshal(sBefore)
	if err != nil {
		return err
	}
	r, err := GetDiffPatch(a, b)
	if err != nil {
		return err
	}
	c.rollback = string(r)
	return nil
}

// inferPatch attempts to solve for the Change's patch value.
// This may be needed for rollbacks where we only have before and instructions.
func (c *Change) inferPatch() {

	if len(c.patch) == 0 &&
		(c.command == MigratorAdd || c.command == MigratorSet || c.command == MigratorUpdate) {
		c.patch = c.after
	}

}

// inferPrettyDiff attempts to solve for the Change's prettyDiff value.
func (c *Change) inferPrettyDiff() error {

	if c.before == nil || c.after == nil {
		return errors.New("Need before and after value to infer pretty diff.")
	}

	sBefore, sAfter := c.beforeAfterCache()
	s, err := PrettyDiff(sBefore, sAfter)
	if err != nil {
		return err
	}

	c.prettyDiff = s
	return nil
}

// Present prints the Change's state to stdout.
func (c *Change) Present() {

	fmt.Println("Target:	" + clrTheme().blue(c.docPath) + fmt.Sprintf("	>> [%s]", strings.ToUpper(c.commandString())) + "\n")

	if c.errState != nil {
		fmt.Println("< !!! ERROR STATE !!! >")
		fmt.Println(c.errState.Error())
		return

	} else if len(c.prettyDiff) == 0 {
		fmt.Println("< no changes >")

	} else {
		replace := []string{"__timestamp__", "__delete__", "__docref__"}
		s := c.prettyDiff

		for _, r := range replace {
			s = strings.Replace(s, `"`+r, "", 1)
			s = strings.Replace(s, r+`"`, "", 1)
		}

		fmt.Println(s)
	}
}

// pushChange executes this change unit against the database.
func (c *Change) pushChange(database Firestore, transformer func(map[string]any) map[string]any) error {
	data := transformer(c.patch)
	switch c.command {
	case MigratorUpdate:
		return database.UpdateDoc(c.docPath, data)
	case MigratorSet:
		return database.SetDoc(c.docPath, data)
	case MigratorAdd:
		return database.SetDoc(c.docPath, data)
	default:
		return database.DeleteDoc(c.docPath)
	}
}

func (c *Change) beforeAfterCache() (map[string]any, map[string]any) {
	// var sBefore, sAfter string
	sBefore, ok := c.cache["serialBefore"]
	if !ok {
		sBefore = SerializeData(c.before, c.database).(map[string]any)
		c.cache["serialBefore"] = sBefore
	}
	sAfter, ok := c.cache["serialAfter"]
	if !ok {
		sAfter = SerializeData(c.after, c.database).(map[string]any)
		c.cache["serialAfter"] = sAfter
	}
	return sBefore, sAfter
}
