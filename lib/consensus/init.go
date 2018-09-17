package consensus

import (
	"os"

	logging "github.com/inconshreveable/log15"
)

var log logging.Logger = logging.New("module", "isaac")

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}

func init() {
	SetLogging(logging.LvlCrit, logging.StreamHandler(os.Stdout, logging.TerminalFormat()))
}
