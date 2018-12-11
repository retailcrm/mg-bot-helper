package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/op/go-logging"
)

// Options struct
type Options struct {
	Config string `short:"c" long:"config" default:"config.yml" description:"Path to configuration file"`
}

var (
	config       *BotConfig
	orm          *Orm
	logger       *logging.Logger
	options      Options
	tokenCounter uint32
	parser       = flags.NewParser(&options, flags.Default)
	currency     = map[string]string{
		"Российский рубль":  "rub",
		"Гри́вня":           "uah",
		"Беларускі рубель":  "byr",
		"Қазақстан теңгесі": "kzt",
		"U.S. dollar":       "usd",
		"Euro":              "eur",
	}
)

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
}
