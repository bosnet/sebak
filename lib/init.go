package sebak

import (
	logging "github.com/inconshreveable/log15"
)

var log logging.Logger

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log = logging.New("module", "sebak")
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}
