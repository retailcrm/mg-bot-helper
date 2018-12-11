package main

import (
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// Orm struct
type Orm struct {
	DB *gorm.DB
}

// NewDb init new database connection
func NewDb(config *BotConfig) *Orm {
	db, err := gorm.Open("postgres", config.Database.Connection)
	if err != nil {
		panic(err)
	}

	db.DB().SetConnMaxLifetime(time.Duration(config.Database.ConnectionLifetime) * time.Second)
	db.DB().SetMaxOpenConns(config.Database.MaxOpenConnections)
	db.DB().SetMaxIdleConns(config.Database.MaxIdleConnections)

	db.SingularTable(true)
	db.LogMode(config.Database.Logging)

	return &Orm{
		DB: db,
	}
}

// Close connection
func (orm *Orm) Close() {
	orm.DB.Close()
}
