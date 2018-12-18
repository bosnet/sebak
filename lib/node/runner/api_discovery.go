package runner

import (
	"io/ioutil"
	"net/http"

	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/network/httputils"
)

// DiscoveryHandler will receive the `DiscoveryMessage` and checks the
// undiscovered validators. If found, trying to update validator data.
func (nh NetworkHandlerNode) DiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	dm, err := network.DiscoveryMessageFromJSON(body)
	if err != nil {
		http.Error(w, err.Error(), httputils.StatusCode(err))
		return
	}

	if err := dm.IsWellFormed(nh.conf); err != nil {
		http.Error(w, err.Error(), httputils.StatusCode(err))
		return
	}

	if !nh.localNode.HasValidators(dm.B.Address) {
		err := errors.DiscoveryFromUnknownValidator
		http.Error(w, err.Error(), httputils.StatusCode(err))
		return
	}

	if err := nh.consensus.ConnectionManager().Discovery(dm); err != nil {
		http.Error(w, err.Error(), httputils.StatusCode(err))
		return
	}
}
