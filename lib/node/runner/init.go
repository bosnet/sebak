package runner

import (
	"time"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
)

var log logging.Logger = logging.New("module", "noderunner")
var DebugPProf bool = false
var startTime time.Time

func init() {
	SetLogging(common.DefaultLogLevel, common.DefaultLogHandler)
	startTime = time.Now()
}

func SetLogging(level logging.Lvl, handler logging.Handler) {
	log.SetHandler(logging.LvlFilterHandler(level, handler))
}
