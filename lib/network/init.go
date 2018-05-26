package network

import (
	"os"

	logging "github.com/inconshreveable/log15"
	"github.com/spikeekips/sebak/lib/util"
)

var log logging.Logger = logging.New("module", "network")

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}

func init() {
	if util.InTestVerbose() {
		SetLogging(logging.LvlDebug, logging.StreamHandler(os.Stdout, logging.TerminalFormat()))
	} else {
		SetLogging(logging.LvlCrit, logging.StreamHandler(os.Stdout, logging.TerminalFormat()))
	}
}
