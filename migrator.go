package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"
)

type RollbackUnit struct {
	DocPath string `json:"docPath"`
	Instruction string `json:"instruction"`
}

type Rollback struct {
	DatabaseName string `json:"databaseName"`
	Timestamp time.Time `json:"timestamp"`
	ChangeUnits []RollbackUnit `json:"changeUnits"`
	Executed bool
}

// <---------------------- Migrator ------------------------------------>

type Migrator struct {
	name string
	storagePath string
	deleteFlag  string
	database    Firestore
	changes     []*Change
	isRollback  bool
}

func NewMigrator(storagePath string, database Firestore, name string) *Migrator {
	m := Migrator{
		name: name,
		storagePath: storagePath,
		deleteFlag:  "<delete>",
		database:    database,
	}
	return &m
}

func (m *Migrator) buildRollback() (*Rollback, error) {
	rollback := Rollback{
		DatabaseName: m.database.Name(),
		Timestamp: time.Now(),
		Executed: false,
	}
	for _, c := range m.changes {
		if c.errState != nil {
			return nil, errors.New("Detected error state on changes.")
		}
		u := RollbackUnit{
			DocPath: c.docPath,
			Instruction: c.rollback,
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
    js, err := json.Marshal(rollback)
	if err != nil {
		return err
	}
    err = ioutil.WriteFile(m.storagePath + "/" + m.name + ".json", js, 0644)
	return err
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
}

// TODO
func (m *Migrator) LoadRollback() {}

type Stager struct {
	migrator *Migrator
}

func (s Stager) Update(docPath string, data map[string]any) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorUpdate)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Set(docPath string, data map[string]any) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorSet)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Add(colPath string, data map[string]any) error {
	path, err := s.migrator.database.GenDocPath(colPath)
	if err != nil {
		return err
	}
	change := NewChange(path, map[string]any{}, data, MigratorSet)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Delete(docPath string) error {
	before, err := s.migrator.database.GetDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, map[string]any{}, MigratorSet)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}

func (m *Migrator) Stage() *Stager {
	s := Stager{
		migrator: m,
	}
	return &s
}
