package main

// controller is meant to orchestrate Migrator functionality.
type controller struct {
	mig    *Migrator
	config map[string]string
}

// NewController is a controller factory.
func NewController(config map[string]string) *controller {
	c := controller{
		config: config,
		mig:    new(Migrator), // TODO
	}
	return &c
}

// TODO
func (c controller) manageMigration() {
	// inject firestore and boot migrator
}
