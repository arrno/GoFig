package main

import (
	"fmt"
	"strings"
)

// controller is meant to orchestrate Migrator functionality.
type Controller struct {
	mig    *Migrator
	config Config
	Close  func()
}

// Config is the expected structure for controller config
type Config struct {
	keyPath     string
	storagePath string
	name        string
}

// NewController is a controller factory. Defer *controller.Close() after initialization.
func NewController(config Config) (*Controller, error) {
	ff, close, err := NewFirestore(config.keyPath)
	if err != nil {
		return nil, err
	}
	mig := NewMigrator(config.storagePath, ff, config.name)
	c := Controller{
		config: config,
		mig:    mig,
		Close:  close,
	}
	return &c, nil
}

// Stage exposes the Migrator Stager as an API to the end user.
func (c *Controller) Stage() *Stager {
	return c.mig.Stage()
}

// ManageStagedMigration runs the CLI script for handling a Migration that has been staged.
// If the migration is to be loaded from a migration file in storage, set load to true.
func (c *Controller) ManageStagedMigration(load bool) {

	ClearTerm()
	if load {
		if err := c.mig.LoadMigration(); err != nil {
			fmt.Println("LoadError: " + err.Error())
			return
		}
	}
	if err := c.prepAndPresent(false); err != nil {
		fmt.Println("PrepError: " + err.Error())
		return
	}
	c.promptRun()

}

// prepAndPresent is a script to prepare the migration and present it via stdout.
func (c *Controller) prepAndPresent(clear bool) error {
	if clear {
		ClearTerm()
	}
	if err := c.mig.PrepMigration(); err != nil {
		return err
	}
	c.mig.PresentMigration()
	return nil
}

// promptRun is a script to prompt a user for confirmation. If the user confirms
// in the affirmative, the migration is run against the database.
func (c *Controller) promptRun() {
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
func (c *Controller) DeleteField() any {
	return c.mig.database.DeleteField()
}

// RefField is a shortcut to the controlled database RefField
func (c *Controller) RefField(docPath string) any {
	return c.mig.database.RefField(docPath)
}
