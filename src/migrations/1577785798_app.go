package migrations

import (
	"errors"

	"github.com/jinzhu/gorm"
	"github.com/retailcrm/mg-transport-core/core"
	"gopkg.in/gormigrate.v1"
)

func init() {
	core.Migrations().Add(&gormigrate.Migration{
		ID: "1577785798",
		Migrate: func(db *gorm.DB) error {
			if db.HasTable("schema_migrations") {
				return db.DropTable("schema_migrations").Error
			}

			return nil
		},
		Rollback: func(db *gorm.DB) error {
			return errors.New("this migration cannot be rolled back")
		},
	})
}
