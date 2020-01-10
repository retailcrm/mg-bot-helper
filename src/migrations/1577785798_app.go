package migrations

import (
	"github.com/jinzhu/gorm"
	"github.com/retailcrm/mg-transport-core/core"
	"gopkg.in/gormigrate.v1"
)

func init() {
	core.Migrations().Add(&gormigrate.Migration{
		ID: "1577785798",
		Migrate: func(db *gorm.DB) error {
			if db.HasTable("schema_migrations") {
				return db.Exec("ALTER TABLE schema_migrations RENAME TO schema_migrations_old;").Error
			}

			return nil
		},
		Rollback: func(db *gorm.DB) error {
			if db.HasTable("schema_migrations_old") {
				return db.Exec("ALTER TABLE schema_migrations_old RENAME TO schema_migrations;").Error
			}

			return nil
		},
	})
}
