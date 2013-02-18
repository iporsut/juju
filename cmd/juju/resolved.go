package main

import (
	"fmt"
	"launchpad.net/gnuflag"
	"launchpad.net/juju-core/cmd"
	"launchpad.net/juju-core/juju"
	"launchpad.net/juju-core/state"
)

// ResolvedCommand marks a unit in an error state as ready to continue.
type ResolvedCommand struct {
	EnvName  string
	UnitName string
	Retry    bool
}

func (c *ResolvedCommand) Info() *cmd.Info {
	return &cmd.Info{"resolved", "<unit>", "marks unit errors resolved", ""}
}

func (c *ResolvedCommand) SetFlags(f *gnuflag.FlagSet) {
	addEnvironFlags(&c.EnvName, f)
	f.BoolVar(&c.Retry, "r", false, "re-execute failed hooks")
	f.BoolVar(&c.Retry, "retry", false, "")
}

func (c *ResolvedCommand) Init(f *gnuflag.FlagSet, args []string) error {
	if err := f.Parse(true, args); err != nil {
		return err
	}
	args = f.Args()
	if len(args) > 0 {
		c.UnitName = args[0]
		if !state.IsUnitName(c.UnitName) {
			return fmt.Errorf("invalid unit name %q", c.UnitName)
		}
		args = args[1:]
	} else {
		return fmt.Errorf("no unit specified")
	}
	return cmd.CheckEmpty(args)
}

func (c *ResolvedCommand) Run(_ *cmd.Context) error {
	conn, err := juju.NewConnFromName(c.EnvName)
	if err != nil {
		return err
	}
	defer conn.Close()
	unit, err := conn.State.Unit(c.UnitName)
	if err != nil {
		return err
	}
	return conn.Resolved(unit, c.Retry)
}
