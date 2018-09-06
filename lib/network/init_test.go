package network

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common/test"
)

func init() {
	SetLogging(logging.LvlDebug, test.LogHandler())
}
