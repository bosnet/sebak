package runner

import (
	"boscoin.io/sebak/lib/common"
	logging "github.com/inconshreveable/log15"
)

var log logging.Logger = logging.New("module", "noderunner")

func init() {
	SetLogging(common.DefaultLogLevel, common.DefaultLogHandler)
}

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}
