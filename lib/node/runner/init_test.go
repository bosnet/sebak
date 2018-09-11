package runner

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/test"
)

func init() {
	common.SetLogging(log, logging.LvlDebug, test.LogHandler())
}
