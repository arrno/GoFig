package main

import (
	"encoding/json"
	"errors"
)

// <---------------------- Change ------------------------------------>
type Command int

const (
	MigratorUpdate Command = iota
	MigratorSet
	MigratorAdd
	MigratorDelete
)

type Change struct {
	docPath    string
	before     map[string]any
	patch      map[string]any
	after      map[string]any
	command    Command
	prettyDiff string
	rollback   string
}

func NewChange(docPath string, before map[string]any, patch map[string]any, command Command) *Change {
	c := Change{
		docPath: docPath,
		before:  before,
		patch:   patch,
		command: command,
	}
	return &c
}

func (c *Change) solveChange() {
	c.inferAfter()
	c.inferPrettyDiff()
	c.inferRollback()
}

func (c *Change) commandString() string {
	switch c.command {
	case 1:
		return "update"
	case 2:
		return "set"
	case 3:
		return "add"
	default:
		return "delete"
	}
}

func (c *Change) inferAfter() error {
	if c.command != 0 {
		switch c.command {
		case 2:
			c.after = c.patch
			return nil
		case 3:
			c.after = c.patch
			return nil
		case 4:
			c.after = map[string]any{}
			return nil
		}
	}
	if c.before == nil || c.after == nil {
		return errors.New("Need before and patch to infer after.")
	}
	bm, err := json.Marshal(c.before)
	if err != nil {
		return err
	}
	pm, err := json.Marshal(c.patch)
	if err != nil {
		return err
	}
	after, err := applyDiffPatch(bm, pm)
	if err != nil {
		return err
	}
	var ua map[string]any
	json.Unmarshal(after, &ua)
	c.after = ua
	return nil
}

func (c *Change) inferCommand() error {
	// this is only really needed for rollbacks

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

func (c *Change) inferRollback() error {
	if c.before == nil || c.after == nil {
		return errors.New("Need before and after value to infer rollback.")
	}
	a, err := json.Marshal(c.after)
	if err != nil {
		return err
	}
	b, err := json.Marshal(c.before)
	if err != nil {
		return err
	}
	r, err := getDiffPatch(a, b)
	if err != nil {
		return err
	}
	c.rollback = string(r)
	return nil
}

func (c *Change) inferPrettyDiff() error {

	if c.before == nil || c.after == nil {
		return errors.New("Need before and after value to infer pretty diff.")
	}

	s, err := prettydiff(c.before, c.after)
	if err != nil {
		return err
	}

	c.prettyDiff = s
	return nil
}

func (c *Change) pushChange(database Firestore) error {
	switch c.command {
	case 1:
		return database.updateDoc(c.docPath, c.patch)
	case 2:
		return database.setDoc(c.docPath, c.patch)
	case 3:
		return database.setDoc(c.docPath, c.patch)
	default:
		return database.deleteDoc(c.docPath)
	}
}
