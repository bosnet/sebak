package sebak

import (
	"boscoin.io/sebak/lib/common"
	logging "github.com/inconshreveable/log15"
)

var log logging.Logger = logging.New("module", "sebak")

func init() {
	common.SetLogging(log, common.DefaultLogLevel, common.DefaultLogHandler)
}

func SetLogging(level logging.Lvl, handler logging.Handler) {
	common.SetLogging(log, level, handler)
}
