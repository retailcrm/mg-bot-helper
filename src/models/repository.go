package models

import (
	"github.com/retailcrm/mg-transport-core/core"
	"gopkg.in/yaml.v2"
)

var app *core.Engine

// SetApp set app for all models
func SetApp(engine *core.Engine) {
	if engine == nil {
		panic("engine shouldn't be nil")
	}

	app = engine
}

// GetConfig returns configuration
func GetConfig() *Config {
	return app.Config.(*Config)
}

// LoadConfig read & load configuration file
func (c *Config) LoadConfig(path string) *Config {
	data := c.GetConfigData(path)
	c.Config.LoadConfigFromData(data)
	if err := yaml.Unmarshal(data, c); err != nil {
		panic(err)
	}

	return c
}

func GetConnection(uid string) *Connection {
	var connection Connection
	app.DB.First(&connection, "client_id = ?", uid)

	return &connection
}

func GetConnectionByURL(urlCrm string) *Connection {
	var connection Connection
	app.DB.First(&connection, "api_url = ?", urlCrm)

	return &connection
}

func GetActiveConnection() []*Connection {
	var connection []*Connection
	app.DB.Find(&connection, "active = ?", true)

	return connection
}

func (c *Connection) SetConnectionActivity() error {
	return app.DB.Model(c).Where("client_id = ?", c.ClientID).Updates(map[string]interface{}{"active": c.Active, "api_url": c.URL}).Error
}

func (c *Connection) SaveConnection() error {
	return app.DB.Model(c).Where("client_id = ?", c.ClientID).Update(c).Error
}

func (c *Connection) CreateConnection() error {
	return app.DB.Create(c).Error
}

func (c *Connection) NormalizeApiUrl() {
	c.URL = app.RemoveTrailingSlash(c.URL)
}
