package main

import "fmt"

// <---------------------- Migrator ------------------------------------>
type Migrator struct {
	keyPath     string
	storagePath string
	deleteFlag  string
	database    Firestore
	changes     []*Change
	isRollback  bool
}

func NewMigrator(keyPath string, storagePath string, database Firestore) *Migrator {
	m := Migrator{
		keyPath:     keyPath,
		storagePath: storagePath,
		deleteFlag:  "<delete>",
		database: database,
	}
	return &m
}

// TODO
func (m *Migrator) storeRollback()   {}
func (m *Migrator) validateWorkset() {}

func (m *Migrator) SetDeleteFlag(flag string) {
	m.deleteFlag = flag
}
// TODO
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
			fmt.Println(err.Error()+"\n")
		}
	}
}
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

