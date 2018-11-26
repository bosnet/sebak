package network

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
)

var VerboseLogs bool
var log logging.Logger = logging.New("module", "network")
var httpLog logging.Logger = logging.New("module", "http")

func init() {
	SetLogging(common.DefaultLogLevel, common.DefaultLogHandler)
	SetHTTPLogging(common.DefaultLogLevel, common.DefaultLogHandler)
}

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}

func SetHTTPLogging(level logging.Lvl, handler logging.Handler) {
	httpLog.SetHandler(logging.LvlFilterHandler(level, handler))
}
