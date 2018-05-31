package sebak

import (
	"os"

	logging "github.com/inconshreveable/log15"
)

var networkID []byte = []byte("sebak-test-network")

func init() {
	SetLogging(logging.LvlDebug, logging.StreamHandler(os.Stdout, logging.TerminalFormat()))
}
