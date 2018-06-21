package sebak

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common/test"
)

var networkID []byte = []byte("sebak-test-network")

func init() {
	SetLogging(logging.LvlDebug, test.LogHandler())
}
