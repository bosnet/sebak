package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/observer"
	"context"
	"fmt"
	"net/http"
)

const GetNodePattern = "/node"

func GetNodeHandler(ctx context.Context) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var err error

		cn := ctx.Value("currentNode").(sebakcommon.Node)

		switch r.Header.Get("Accept") {
		case "text/event-stream":
			var readyChan = make(chan struct{})
			iterateId := sebakcommon.GetUniqueIDFromUUID()
			go func() {
				<-readyChan
				observer.NodeObserver.Trigger(fmt.Sprintf("iterate-%s", iterateId), cn)
			}()

			event := "change"
			event += " " + fmt.Sprintf("iterate-%s", iterateId)
			callBackFunc := func(args ...interface{}) (btSerialized []byte, err error) {
				cn := args[1].(sebakcommon.Node)
				var cnSerialized []byte
				if cnSerialized, err = cn.Serialize(); err != nil {
					return []byte{}, sebakerror.ErrorSerialized
				}
				return cnSerialized, nil
			}
			streaming(observer.NodeObserver, w, event, callBackFunc, readyChan)
		default:
			var cnSerialized []byte
			if cnSerialized, err = cn.Serialize(); err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}

			if _, err = w.Write(cnSerialized); err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
		}
	}
}
