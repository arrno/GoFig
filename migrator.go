package main

import (
	"errors"
	"fmt"
	"time"
)

type WorkUnit struct {
	DocPath     string         `json:"docPath"`
	Instruction string         `json:"instruction,omitempty"`
	Patch       map[string]any `json:"patch,omitempty"`
	Command     Command        `json:"command,omitempty"`
}

type Migration struct {
	DatabaseName string     `json:"databaseName"`
	Timestamp    time.Time  `json:"timestamp"`
	ChangeUnits  []WorkUnit `json:"changeUnits"`
	Executed     bool
}

// <---------------------- Migrator ------------------------------------>

type Migrator struct {
	name        string
	storagePath string
	deleteFlag  string
	database    Firestore
	changes     []*Change
	hasRun      bool
}

func NewMigrator(storagePath string, database Firestore, name string) *Migrator {
	m := Migrator{
		name:        name,
		storagePath: storagePath,
		deleteFlag:  "<delete>",
		database:    database,
	}
	return &m
}

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
			command = MigratorUnknown
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

func (m *Migrator) storeRollback() error {
	rollback, err := m.buildRollback()
	if err != nil {
		return err
	}
	return storeJson(rollback, m.storagePath, m.name+"_rollback")
}

// TODO
func (m *Migrator) validateWorkset() {}

func (m *Migrator) SetDeleteFlag(flag string) {
	m.deleteFlag = flag
}

func (m *Migrator) CrunchMigration() {
	for _, c := range m.changes {
		c.SolveChange()
	}
}
func (m *Migrator) PresentMigration() {
	for _, c := range m.changes {
		c.Present()
		fmt.Println("\n<--------------------------------------------------->")
		fmt.Println("<--------------------------------------------------->\n")
	}
}
func (m *Migrator) RunMigration() {
	for _, c := range m.changes {
		err := c.pushChange(m.database)
		if err != nil {
			fmt.Println(c.docPath)
			fmt.Println("\n< ERROR EXEC... error on change execution. >")
			fmt.Println(err.Error() + "\n")
		}
	}
	m.hasRun = true
	m.StoreMigration()
}

func (m *Migrator) LoadMigration() error {
	var mig Migration
	err := loadJson(m.storagePath+"/"+m.name, &mig)
	if err != nil {
		return err
	}
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
	m.CrunchMigration()
	return nil
}

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

	return storeJson(migration, m.storagePath, m.name)

}

type Stager struct {
	migrator *Migrator
}

func (s Stager) Update(docPath string, data map[string]any, instruction string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorUpdate, instruction)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Set(docPath string, data map[string]any, instruction string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorSet, instruction)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Add(colPath string, data map[string]any, instruction string) error {
	path, err := s.migrator.database.GenDocPath(colPath)
	if err != nil {
		return err
	}
	change := NewChange(path, map[string]any{}, data, MigratorSet, instruction)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Delete(docPath string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, map[string]any{}, MigratorDelete, "")
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Unknown(docPath string, data map[string]any, instruction string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorUnknown, instruction)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

func (m *Migrator) Stage() *Stager {
	s := Stager{
		migrator: m,
	}
	return &s
}
