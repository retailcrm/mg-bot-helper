package migrations

import (
	"github.com/jinzhu/gorm"
	"github.com/retailcrm/mg-bot-helper/src/models"
	"github.com/retailcrm/mg-transport-core/core"
	"gopkg.in/gormigrate.v1"
)

func init() {
	core.Migrations().Add(&gormigrate.Migration{
		ID: "1577787226",
		Migrate: func(db *gorm.DB) error {
			return db.AutoMigrate(models.Connection{}).Error
		},
		Rollback: func(db *gorm.DB) error {
			return nil
		},
	})
}
