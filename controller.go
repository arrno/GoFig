package main

type controller struct {
	mig *Migrator
	config map[string]string
}

func NewController(config map[string]string) *controller {
	c := controller{
		config: config,
		mig: new(Migrator), // TODO
	}
	return &c
}

func (c controller) manageMigration() {
	// 
}