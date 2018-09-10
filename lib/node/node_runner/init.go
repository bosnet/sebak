package node_runner

import (
	"os"

	logging "github.com/inconshreveable/log15"
)

var log logging.Logger = logging.New("module", "node_runner")

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}

func init() {
	SetLogging(logging.LvlCrit, logging.StreamHandler(os.Stdout, logging.TerminalFormat()))
}
