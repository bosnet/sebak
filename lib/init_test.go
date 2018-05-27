package sebak

import (
	"os"

	logging "github.com/inconshreveable/log15"
)

func init() {
	SetLogging(logging.LvlDebug, logging.StreamHandler(os.Stdout, logging.TerminalFormat()))
}
