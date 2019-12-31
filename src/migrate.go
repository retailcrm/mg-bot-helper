package main

import (
	"fmt"

	_ "github.com/retailcrm/mg-bot-helper/src/migrations"
	"github.com/retailcrm/mg-transport-core/core"
)

func init() {
	_, err := parser.AddCommand("migrate",
		"Migrate database to defined migrations version",
		"Migrate database to defined migrations version.",
		&MigrateCommand{},
	)

	if err != nil {
		panic(err.Error())
	}
}

// MigrateCommand struct
type MigrateCommand struct {
	Version string `short:"v" long:"version" default:"up" description:"Migrate to defined migrations version. Allowed: up, down, next, prev and migration version."`
}

func (x *MigrateCommand) Execute(args []string) error {
	core.Migrations().SetDB(app.DB)

	if err := Migrate(x.Version); err != nil {
		return err
	}

	return nil
}

func Migrate(version string) error {
	currentVersion := core.Migrations().Current()

	defer core.Migrations().Close()

	if "up" == version {
		fmt.Printf("Migrating from %s to last\n", currentVersion)
		return core.Migrations().Migrate()
	}

	if "down" == version {
		fmt.Printf("Migrating from %s to 0\n", currentVersion)
		return core.Migrations().Rollback()
	}

	if "next" == version {
		fmt.Printf("Migrating from %s to next", currentVersion)
		return core.Migrations().MigrateNextTo(currentVersion)
	}

	if "prev" == version {
		fmt.Printf("Migration from %s to previous", currentVersion)
		return core.Migrations().MigratePreviousTo(currentVersion)
	}

	fmt.Printf("Migration from %s to %s", currentVersion, version)
	return core.Migrations().MigrateTo(version)
}
