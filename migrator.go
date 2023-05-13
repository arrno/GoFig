package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// WorkUnit is one change mapped to one document within a migration.
// You cannot have multiple work units pointing to the same document in
// a migration.
type WorkUnit struct {
	DocPath string         `json:"docPath"`
	Patch   map[string]any `json:"patch,omitempty"`
	Command Command        `json:"command,omitempty"`
}

// Migration represents all the instructions needed by the migrator to orchestrate a job.
// All migration jobs including rollbacks take this form.
type Migration struct {
	DatabaseName string     `json:"databaseName"`
	Timestamp    time.Time  `json:"timestamp"`
	ChangeUnits  []WorkUnit `json:"changeUnits"`
	Executed     bool       `json:"executed"`
}

// <---------------------- Migrator ------------------------------------>

// Migrator is the API for performing migration tasks within a job.
type Migrator struct {
	name        string
	storagePath string
	deleteFlag  string
	database    Firestore
	changes     []*Change
	hasRun      bool
}

// NewMigrator is a Migrator factory.
func NewMigrator(storagePath string, database Firestore, name string) *Migrator {
	m := Migrator{
		name:        name,
		storagePath: storagePath,
		deleteFlag:  "<delete>",
		database:    database,
	}
	return &m
}

// buildRollback take the current Migrator state and prodces a Migration struct which
// can later be loaded and run by the Migrator to rollback/inverse the initial state.
func (m *Migrator) buildRollback() (*Migration, error) {
	rollback := Migration{
		DatabaseName: m.database.Name(),
		Timestamp:    time.Now(),
		Executed:     false,
	}
	for _, c := range m.changes {
		if c.errState != nil {
			return nil, errors.New("Detected error state on changes.")
		}
		var command Command
		switch c.command {
		case MigratorAdd:
			command = MigratorDelete
			break
		case MigratorUpdate:
			command = MigratorUpdate
			break
		case MigratorDelete:
			command = MigratorAdd
			break
		case MigratorSet:
			if len(c.before) == 0 {
				command = MigratorDelete
			} else {
				command = MigratorUpdate
			}
			break
		default:
			command = MigratorUnknown
		}
		u := WorkUnit{
			DocPath: c.docPath,
			Patch:   SerializeData(c.rollback, m.database).(map[string]any),
			Command: command,
		}
		rollback.ChangeUnits = append(rollback.ChangeUnits, u)
	}
	return &rollback, nil
}

// storeRollback builds a rollback and stores the resulting Migration instructions to storage.
func (m *Migrator) storeRollback() error {
	rollback, err := m.buildRollback()
	if err != nil {
		return err
	}
	return StoreJson(rollback, m.storagePath, m.name+"_rollback")
}

// validateWorkset returns a new error if the staged Changes are not valid
func (m *Migrator) validateWorkset() error {
	// No duplicate docpath refs
	docPaths := map[string]bool{}
	for _, change := range m.changes {
		_, ok := docPaths[change.docPath]
		if ok {
			return errors.New("Cannot have multiple changes staged against the same document reference.")
		}
		docPaths[change.docPath] = true
	}
	return nil
}

// SetDeleteFlag updates the Migrator delete flag with a new string value. When this value
// is on a Change field, that field is deleted from the database document when the change is pushed.
func (m *Migrator) SetDeleteFlag(flag string) {
	m.deleteFlag = flag
}

type TransformMode int

const (
	Serialize TransformMode = iota
	DeSerialize
	Exclude
)

// PrepMigration is run after all changes are staged. This function validates and solves all of the changes.
// No changes are pushed to the database.
func (m *Migrator) PrepMigration() error {
	err := m.validateWorkset()
	if err != nil {
		return err
	}
	for _, c := range m.changes {
		c.SolveChange()
	}
	return nil
}

// PresentMigration prints all the staged changes to stdout for review.
func (m *Migrator) PresentMigration() {

	lngth := MaxNum(len(m.name), len(m.database.Name()))
	lngth = MaxNum(lngth, len(m.storagePath)) + 26
	m.printSeparator(lngth)

	fmt.Printf(
		"Migration Name:	%s\nDatabase:	%s\nStorage Path:	%s\nHas Run:	%v\n",
		"  "+m.name,
		"  "+m.database.Name(),
		"  "+m.storagePath,
		"  "+strconv.FormatBool(m.hasRun),
	)
	for _, c := range m.changes {
		lngth = len(c.docPath) + len(c.commandString()) + 19
		header, cOut := c.Present()
		lineLength, _ := LongestLine(cOut)
		maxLength := MaxNum(lngth, lineLength-12)
		m.printSeparator(maxLength)
		headerPad := strings.Repeat(" ", maxLength-utf8.RuneCountInString(header[0]+header[1])+14)
		fmt.Print(strings.Join(header, headerPad))
		fmt.Print(cOut)
	}
	m.printSeparator(lngth)
}

// printSeparator prints a horizontal separator to stdout
func (m *Migrator) printSeparator(length int) {
	dashes := strings.Repeat("-", length)
	// 50
	fmt.Printf("\n<%s>\n", dashes)
	fmt.Printf("<%s>\n\n", dashes)
}

// RunMigration executes all of the staged changes against the database.
func (m *Migrator) RunMigration() {
	for _, c := range m.changes {
		err := c.pushChange(
			func(data map[string]any) map[string]any {
				return data
				// TODO
				// return m.toggleDeleteFlag(data, DeSerialize)
			},
		)
		if err != nil {
			fmt.Println("\n< !!! EXECUTION ERROR !!! >")
			fmt.Println(c.docPath)
			fmt.Println(err.Error() + "\n")
		}
	}
	m.hasRun = true
	m.StoreMigration()
	m.storeRollback()
}

// LoadMigration will look for an existing migration file matching this Migrator's name.
// If a file exists, the state of the migrator will be replaced by the contents of the file.
// This is the preferred workflow for loading a rollback.
func (m *Migrator) LoadMigration() error {
	var mig Migration
	err := LoadJson(m.storagePath+"/"+m.name, &mig)
	if err != nil {
		return err
	}
	m.hasRun = mig.Executed
	m.changes = []*Change{}
	for _, unit := range mig.ChangeUnits {
		patch := DeSerializeData(unit.Patch, m.database).(map[string]any)
		switch unit.Command {
		case MigratorAdd:
			err = m.Stage().Set(unit.DocPath, patch)
			break
		case MigratorSet:
			err = m.Stage().Set(unit.DocPath, patch)
			break
		case MigratorUpdate:
			err = m.Stage().Update(unit.DocPath, patch)
			break
		case MigratorDelete:
			err = m.Stage().Delete(unit.DocPath)
			break
		default:
			err = m.Stage().Unknown(unit.DocPath, patch)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// StoreMigration converts the Migrator state to a Migration file and stores it to disc.
func (m *Migrator) StoreMigration() error {

	migration := Migration{
		DatabaseName: m.database.Name(),
		Timestamp:    time.Now(),
		Executed:     m.hasRun,
	}
	for _, c := range m.changes {
		if c.errState != nil {
			return errors.New("Detected error state on changes.")
		}
		u := WorkUnit{
			DocPath: c.docPath,
			Patch:   SerializeData(c.patch, m.database).(map[string]any),
			Command: c.command,
		}
		migration.ChangeUnits = append(migration.ChangeUnits, u)
	}

	return StoreJson(migration, m.storagePath, m.name)

}

// Stager is an abstraction on top of Migrator which is used as an API
// to stage new Change units on the Migrator.
type Stager struct {
	migrator *Migrator
}

// Update stages a new Update change on the Migrator.
func (s Stager) Update(docPath string, data map[string]any) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorUpdate, s.migrator.database)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

// Set stages a new Set change on the Migrator.
func (s Stager) Set(docPath string, data map[string]any) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorSet, s.migrator.database)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

// Add stages a new Add change on the Migrator.
func (s Stager) Add(colPath string, data map[string]any) error {
	path, err := s.migrator.database.GenDocPath(colPath)
	if err != nil {
		return err
	}
	change := NewChange(path, map[string]any{}, data, MigratorSet, s.migrator.database)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

// Delete stages a new Delete change on the Migrator.
func (s Stager) Delete(docPath string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, map[string]any{}, MigratorDelete, s.migrator.database)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

// Unknown stages a new change on the Migrator of an Unknown command type.
func (s Stager) Unknown(docPath string, data map[string]any) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorUnknown, s.migrator.database)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

// Stage is a Stager factory
func (m *Migrator) Stage() *Stager {
	s := Stager{
		migrator: m,
	}
	return &s
}
