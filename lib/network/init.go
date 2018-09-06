package network

import (
	"os"

	logging "github.com/inconshreveable/log15"
)

var log logging.Logger = logging.New("module", "network")

func SetLogging(level logging.Lvl, handler logging.Handler) {
	lh := logging.LvlFilterHandler(level, handler)
	log.SetHandler(lh)
}

func init() {
	SetLogging(logging.LvlCrit, logging.StreamHandler(os.Stdout, logging.TerminalFormat()))
}
