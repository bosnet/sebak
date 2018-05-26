package sebak

import (
	"os"

	logging "github.com/inconshreveable/log15"
	"github.com/spikeekips/sebak/lib/common"
)

var log logging.Logger = logging.New("module", "sebak")

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}

func init() {
	if sebakcommon.InTestVerbose() {
		SetLogging(logging.LvlDebug, logging.StreamHandler(os.Stdout, logging.TerminalFormat()))
	} else {
		SetLogging(logging.LvlCrit, logging.StreamHandler(os.Stdout, logging.TerminalFormat()))
	}
}
