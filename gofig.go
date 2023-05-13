package fig

import (
	"errors"
	"fmt"
	"strings"
)

// Fig represents what a GoFig manager should do.
type Fig interface {
	Close()
	Stage() FigStager
	LoadFromFile() error
	SaveToFile() error
	ManageStagedMigration()
	DeleteField() any
	RefField(docPath string) any
}

// GoFig is meant to orchestrate Migrator functionality.
type GoFig struct {
	mig    FigMigrator
	config Config
	close  func()
}

// Config is the expected structure for GoFig config
type Config struct {
	KeyPath     string
	StoragePath string
	Name        string
}

// New is a GoFig factory. Defer *GoFig.Close() after initialization.
func New(config Config) (Fig, error) {
	ff, close, err := newFirestore(config.KeyPath)
	if err != nil {
		return nil, err
	}
	mig := NewMigrator(config.StoragePath, ff, config.Name)
	c := GoFig{
		config: config,
		mig:    mig,
		close:  close,
	}
	return &c, nil
}

// Close should be deferred on initialization to handle any database session cleanup.
func (c *GoFig) Close() {
	c.close()
}

// Stage exposes the Migrator Stager as an API to the end user.
func (c *GoFig) Stage() FigStager {
	return c.mig.Stage()
}

// LoadFromFile attempts to load a pre staged migration from a file if it exists
// in the storagePath folder
func (c *GoFig) LoadFromFile() error {
	if err := c.mig.LoadMigration(); err != nil {
		return errors.New("LoadError: " + err.Error())
	}
	return nil
}

// SaveToFile attempts to save a migration staged in runtime memory to
// a file in the storagePath folder
func (c *GoFig) SaveToFile() error {
	if err := c.mig.StoreMigration(); err != nil {
		return errors.New("StoreError: " + err.Error())
	}
	return nil
}

// ManageStagedMigration runs the CLI script for handling a Migration that has been staged.
// If the migration is to be loaded from a migration file in storage, set load to true.
func (c *GoFig) ManageStagedMigration() {

	clearTerm()
	if err := c.prepAndPresent(false); err != nil {
		fmt.Println("PrepError: " + err.Error())
		return
	}
	c.promptRun()

}

// prepAndPresent is a script to prepare the migration and present it via stdout.
func (c *GoFig) prepAndPresent(clear bool) error {
	if clear {
		clearTerm()
	}
	if err := c.mig.PrepMigration(); err != nil {
		return err
	}
	c.mig.PresentMigration()
	return nil
}

// promptRun is a script to prompt a user for confirmation. If the user confirms
// in the affirmative, the migration is run against the database.
func (c *GoFig) promptRun() {
	userConfirm := "N"
	fmt.Println("Execute these changes? (y/N):")
	fmt.Scanln(&userConfirm)

	if strings.ToLower(userConfirm) == "y" {
		fmt.Println("Running migration...")
		c.mig.RunMigration()
		fmt.Println("Complete.")
	} else {
		fmt.Println("No changes applied.")
	}
}

// DeleteField is a shortcut to the controlled database DeleteField
func (c *GoFig) DeleteField() any {
	return c.mig.deleteField()
}

// RefField is a shortcut to the controlled database RefField
func (c *GoFig) RefField(docPath string) any {
	return c.mig.refField(docPath)
}
