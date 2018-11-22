package sync

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
)

var log logging.Logger = logging.New()

func init() {
	SetLogging(common.DefaultLogLevel, common.DefaultLogHandler)
}

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}
