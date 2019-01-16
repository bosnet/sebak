package common

import (
	logging "github.com/inconshreveable/log15"
)

var log logging.Logger = logging.New("module", "common")

func init() {
	SetLogging(DefaultLogLevel, DefaultLogHandler)
}

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}
