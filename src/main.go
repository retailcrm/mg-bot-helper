package main

import (
	"os"

	"github.com/gobuffalo/packr/v2"
	"github.com/jessevdk/go-flags"
	"github.com/retailcrm/mg-bot-helper/src/models"
	"github.com/retailcrm/mg-transport-core/core"
)

// Options struct
type Options struct {
	Config func(string) `short:"c" long:"config" default:"models.GetConfig().yml" description:"Path to configuration file"`
}

// DefaultFatalError for panics and other uncatched errors
const DefaultFatalError = "error_save"

var (
	options = Options{
		Config: func(s string) {
			if app == nil {
				initVariables(s)
			}
		},
	}
	app          *core.Engine
	parser       = flags.NewParser(&options, flags.Default)
	static       *packr.Box
	templates    *packr.Box
	translations *packr.Box
	wm           *WorkersManager
	currency     = map[string]string{
		"Российский рубль":  "rub",
		"Гри́вня":           "uah",
		"Беларускі рубель":  "byr",
		"Қазақстан теңгесі": "kzt",
		"U.S. dollar":       "usd",
		"Euro":              "eur",
	}
)

func init() {
	static = packr.New("assets", "./../static")
	templates = packr.New("templates", "./../templates")
	translations = packr.New("translations", "./../translate")
}

func initVariables(configPath string) {
	app = core.New().WithCookieSessions()
	app.Config = (&models.Config{}).LoadConfig(configPath)
	app.DefaultError = DefaultFatalError
	app.TranslationsBox = translations
	app.Prepare()
	// WorkerManager uses app.Config and app.Logger() under the hood - that's why it should be initialized here
	wm = NewWorkersManager()
	models.SetApp(app)
}

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
}
