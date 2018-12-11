package main

import (
	"os"

	"github.com/op/go-logging"
)

var logFormat = logging.MustStringFormatter(
	`%{time:2006-01-02 15:04:05.000} %{level:.4s} => %{message}`,
)

func newLogger() *logging.Logger {
	logger := logging.MustGetLogger(config.BotInfo.Code)
	logBackend := logging.NewLogBackend(os.Stdout, "", 0)
	formatBackend := logging.NewBackendFormatter(logBackend, logFormat)
	backend1Leveled := logging.AddModuleLevel(logBackend)
	backend1Leveled.SetLevel(config.LogLevel, "")
	logging.SetBackend(formatBackend)

	return logger
}
