package main

import (
	"errors"
	"fmt"
	"time"
)

// WorkUnit is one change mapped to one document within a migration.
// You cannot have multiple work units pointing to the same document in
// a migration.
type WorkUnit struct {
	DocPath     string         `json:"docPath"`
	Instruction string         `json:"instruction,omitempty"`
	Patch       map[string]any `json:"patch,omitempty"`
	Command     Command        `json:"command,omitempty"`
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
			DocPath:     c.docPath,
			Instruction: c.rollback,
			Command:     command,
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

// toggleDeleteFlag either serializes or deserializes the deleteFlag values on all staged Change structs.
// The original is not changed rather, a copy is returned.
func (m *Migrator) toggleDeleteFlag(data map[string]any, mode TransformMode) map[string]any {
	var before any
	var after any
	// TODO
	switch mode {
	case Serialize:
		before = m.database.DeleteField()
		after = m.deleteFlag
	case DeSerialize:
		before = m.deleteFlag
		after = m.database.DeleteField()
	case Exclude:
		before = m.deleteFlag
		after = nil
	}
	return Transform(data, before, after).(map[string]any)
}

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
	fmt.Printf(
		"\nMigration Name:	%s\nDatabase:	%s\nStorage Path:	%s\nHas Run:	%v\n",
		m.name,
		m.database.Name(),
		m.storagePath,
		m.hasRun,
	)
	fmt.Println("\n<--------------------------------------------------->")
	fmt.Println("<--------------------------------------------------->\n")
	for _, c := range m.changes {
		c.Present()
		fmt.Println("\n<--------------------------------------------------->")
		fmt.Println("<--------------------------------------------------->\n")
	}
}

// RunMigration executes all of the staged changes against the database.
func (m *Migrator) RunMigration() {
	for _, c := range m.changes {
		err := c.pushChange(
			m.database,
			func(data map[string]any) map[string]any {
				return data
				// TODO
				// return m.toggleDeleteFlag(data, DeSerialize)
			},
		)
		if err != nil {
			fmt.Println(c.docPath)
			fmt.Println("\n< ERROR EXEC... error on change execution. >")
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
		switch unit.Command {
		case MigratorAdd:
			err = m.Stage().Set(unit.DocPath, unit.Patch, unit.Instruction)
			break
		case MigratorSet:
			err = m.Stage().Set(unit.DocPath, unit.Patch, unit.Instruction)
			break
		case MigratorUpdate:
			err = m.Stage().Update(unit.DocPath, unit.Patch, unit.Instruction)
			break
		case MigratorDelete:
			err = m.Stage().Delete(unit.DocPath)
			break
		default:
			err = m.Stage().Unknown(unit.DocPath, unit.Patch, unit.Instruction)
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
			Patch:   c.patch,
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
func (s Stager) Update(docPath string, data map[string]any, instruction string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorUpdate, instruction)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

// Set stages a new Set change on the Migrator.
func (s Stager) Set(docPath string, data map[string]any, instruction string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorSet, instruction)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

// Add stages a new Add change on the Migrator.
func (s Stager) Add(colPath string, data map[string]any, instruction string) error {
	path, err := s.migrator.database.GenDocPath(colPath)
	if err != nil {
		return err
	}
	change := NewChange(path, map[string]any{}, data, MigratorSet, instruction)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

// Delete stages a new Delete change on the Migrator.
func (s Stager) Delete(docPath string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, map[string]any{}, MigratorDelete, "")
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

// Unknown stages a new change on the Migrator of an Unknown command type.
func (s Stager) Unknown(docPath string, data map[string]any, instruction string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorUnknown, instruction)
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
