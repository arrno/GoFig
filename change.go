package gofig

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
	docPath    string
	before     map[string]any
	patch      map[string]any
	after      map[string]any
	command    Command
	prettyDiff string
	rollback   map[string]any
	errState   error
	database   figFirestore
	cache      map[string]map[string]any
}

// NewChange is a Change factory.
func NewChange(docPath string,
	before map[string]any,
	patch map[string]any,
	command Command,
	database figFirestore) *Change {
	c := Change{
		docPath:  docPath,
		before:   before,
		patch:    patch,
		command:  command,
		database: database,
		errState: errors.New("Change has not yet been solved."),
		cache:    map[string]map[string]any{},
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
	if c.before == nil || c.patch == nil {
		return errors.New("Need before and patch to infer after.")
	}

	sBefore := c.fetchCache("sBefore", c.before)
	sPatch := c.fetchCache("sPatch", c.patch)
	bm, err := json.Marshal(sBefore)
	if err != nil {
		return err
	}

	var pm []byte
	pm, err = json.Marshal(sPatch)
	if err != nil {
		return err
	}
	after, err := applyDiffPatch(bm, pm)
	if err != nil {
		return err
	}

	var ua map[string]any
	json.Unmarshal(after, &ua)
	c.after = deSerializeData(ua, c.database).(map[string]any)
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

	if len(c.after) > 0 {
		mafter, _ := json.Marshal(c.after)
		mpatch, _ := json.Marshal(c.patch)
		if string(mafter) != string(mpatch) {
			c.command = MigratorUpdate
			return nil
		}
		c.command = MigratorSet
	} else {
		c.command = MigratorDelete
	}

	return nil
}

// inferRollback attempts to solve for the Change's rollback value.
// The rollback value is in the form of json patch
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
	patch, err := getDiffPatch(a, b)
	if err != nil {
		return err
	}
	json.Unmarshal(patch, &c.rollback)
	return nil
}

// inferPrettyDiff attempts to solve for the Change's prettyDiff value.
func (c *Change) inferPrettyDiff() error {

	if c.before == nil || c.after == nil {
		return errors.New("Need before and after value to infer pretty diff.")
	}

	sBefore, sAfter := c.beforeAfterCache()
	s, err := prettyDiff(sBefore, sAfter)
	if err != nil {
		return err
	}

	c.prettyDiff = s
	return nil
}

// Present returns a pretty representation of the change for printing to stdout.
func (c *Change) Present() ([]string, string) {
	out := ""
	header := []string{"Target: " + clrTheme().blue(c.docPath), fmt.Sprintf(" >> [%s]", strings.ToUpper(c.commandString())) + "\n\n"}

	if c.errState != nil {
		out += fmt.Sprintf("< !!! ERROR STATE !!! >\n")
		out += fmt.Sprintf(c.errState.Error() + "\n")
		return header, out

	} else if len(c.prettyDiff) == 0 {
		out += fmt.Sprintf("< no changes >\n")

	} else {
		replace := []string{"__timestamp__", "__delete__", "__docref__"}
		s := c.prettyDiff

		for _, r := range replace {
			s = strings.Replace(s, `"`+r, "", -1)
			s = strings.Replace(s, r+`"`, "", -1)
		}

		out += fmt.Sprintf(s + "\n")
	}
	return header, out

}

// pushChange executes this change unit against the database.
func (c *Change) pushChange(transformer func(map[string]any) map[string]any) error {
	data := transformer(c.patch)
	switch c.command {
	case MigratorUpdate:
		return c.database.updateDoc(c.docPath, data)
	case MigratorSet:
		return c.database.setDoc(c.docPath, data)
	case MigratorAdd:
		return c.database.setDoc(c.docPath, data)
	default:
		return c.database.deleteDoc(c.docPath)
	}
}

// fetchCache returns value from cache
func (c *Change) fetchCache(key string, data map[string]any) map[string]any {
	sVar, ok := c.cache[key]
	if !ok {
		sVar = serializeData(data, c.database).(map[string]any)
		c.cache[key] = sVar
	}
	return sVar
}

// beforeAfterCache returns before and after values from cache
func (c *Change) beforeAfterCache() (map[string]any, map[string]any) {
	// var sBefore, sAfter string
	return c.fetchCache("serialBefore", c.before), c.fetchCache("serialAfter", c.after)
}
