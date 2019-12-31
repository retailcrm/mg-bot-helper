package models

import (
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/retailcrm/mg-transport-core/core"
)

// Connection model
type Connection struct {
	core.Connection
	Commands postgres.Jsonb `gorm:"commands type:jsonb;" json:"commands,omitempty"`
	Lang     string         `gorm:"lang type:varchar(2)" json:"lang,omitempty"`
	Currency string         `gorm:"currency type:varchar(12)" json:"currency,omitempty"`
}

// Config struct
type Config struct {
	core.Config
	BotInfo BotInfo `yaml:"bot_info"`
}

type BotInfo struct {
	Name     string `yaml:"name"`
	Code     string `yaml:"code"`
	LogoPath string `yaml:"logo_path"`
}
