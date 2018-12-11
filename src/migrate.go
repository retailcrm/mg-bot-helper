package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/golang-migrate/migrate"
)

func init() {
	parser.AddCommand("migrate",
		"Migrate database to defined migrations version",
		"Migrate database to defined migrations version.",
		&MigrateCommand{},
	)
}

// MigrateCommand struct
type MigrateCommand struct {
	Version string `short:"v" long:"version" default:"up" description:"Migrate to defined migrations version. Allowed: up, down, next, prev and integer value."`
	Path    string `short:"p" long:"path" default:"" description:"Path to migrations files."`
}

// Execute method
func (x *MigrateCommand) Execute(args []string) error {
	botConfig := LoadConfig(options.Config)

	err := Migrate(botConfig.Database.Connection, x.Version, x.Path)
	if err != nil && err.Error() == "no change" {
		fmt.Println("No changes detected. Skipping migration.")
		err = nil
	}

	return err
}

// Migrate function
func Migrate(database string, version string, path string) error {
	m, err := migrate.New("file://"+path, database)
	if err != nil {
		fmt.Printf("Migrations path %s does not exist or permission denied\n", path)
		return err
	}

	defer m.Close()

	currentVersion, _, err := m.Version()
	if "up" == version {
		fmt.Printf("Migrating from %d to last\n", currentVersion)
		return m.Up()
	}

	if "down" == version {
		fmt.Printf("Migrating from %d to 0\n", currentVersion)
		return m.Down()
	}

	if "next" == version {
		fmt.Printf("Migrating from %d to next\n", currentVersion)
		return m.Steps(1)
	}

	if "prev" == version {
		fmt.Printf("Migrating from %d to previous\n", currentVersion)
		return m.Steps(-1)
	}

	ver, err := strconv.ParseUint(version, 10, 32)
	if err != nil {
		fmt.Printf("Invalid migration version %s\n", version)
		return err
	}

	if ver != 0 {
		fmt.Printf("Migrating from %d to %d\n", currentVersion, ver)
		return m.Migrate(uint(ver))
	}

	fmt.Printf("Migrations not found in path %s\n", path)

	return errors.New("migrations not found")
}
