package main

// <---------------------- Migrator ------------------------------------>
type Migrator struct {
	keyPath     string
	storagePath string
	deleteFlag  string // TODO
	database    Firestore
	changes     []*Change
	isRollback  bool
}

func NewMigrator(keyPath string, storagePath string) *Migrator {
	m := Migrator{
		keyPath:     keyPath,
		storagePath: storagePath,
		deleteFlag:  "<delete>",
		// TODO database 
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
func (m *Migrator) RunMigration() {}
func (m *Migrator) LoadRollback() {}

type Stager struct {
	migrator *Migrator
}

func (s Stager) Update(docPath string, data map[string]any) error {
	before, err := s.migrator.database.getDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorUpdate)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Set(docPath string, data map[string]any) error {
	before, err := s.migrator.database.getDocData(docPath)
	if err != nil {
		return err
	}
	change := NewChange(docPath, before, data, MigratorSet)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Add(data map[string]any) error {
	path, err := s.migrator.database.genDocPath()
	if err != nil {
		return err
	}
	change := NewChange(path, map[string]any{}, data, MigratorSet)
	s.migrator.changes = append(s.migrator.changes, change)
	return nil
}
func (s Stager) Delete(docPath string) error {
	before, err := s.migrator.database.getDocData(docPath)
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

