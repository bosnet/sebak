package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction/operation"
)

func (api NetworkHandlerAPI) GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]

	readFunc := func() (payload interface{}, err error) {
		found, err := block.ExistsBlockAccount(api.storage, address)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.BlockAccountDoesNotExists
		}
		ba, err := block.GetBlockAccount(api.storage, address)
		if err != nil {
			return nil, err
		}
		payload = resource.NewAccount(ba)
		return payload, nil
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("address-%s", address)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		payload, err := readFunc()
		if err == nil {
			es.Render(payload)
		}
		es.Run(observer.BlockAccountObserver, event)
		return
	}

	payload, err := readFunc()
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	httputils.MustWriteJSON(w, 200, payload)
}

func (api NetworkHandlerAPI) GetFrozenAccountsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]
	options, err := storage.NewDefaultListOptionsFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, errors.InvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	var cursor []byte
	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockOperationsByLinked(api.storage, address, options)
		for {
			var (
				createdBlockHeight        uint64
				createdOpHash             string
				sequenceid                uint64
				amount                    common.Amount
				state                     resource.FrozenAccountState
				unfreezingBlockHeight     uint64
				unfreezingOpHash          string
				unfreezingRemainingBlocks uint64
				paymentOpHash             string
			)

			bo, hasNext, c := iterFunc()
			cursor = c
			if !hasNext {
				break
			}
			var (
				casted operation.CreateAccount
				ok     bool
				body   operation.Body
			)
			if body, err = operation.UnmarshalBodyJSON(bo.Type, bo.Body); err != nil {
				break
			}
			if casted, ok = body.(operation.CreateAccount); !ok {
				break
			}
			createdBlockHeight = bo.Height
			createdOpHash = bo.OpHash
			var tx block.BlockTransaction
			if tx, err = block.GetBlockTransaction(api.storage, bo.TxHash); err != nil {
				break
			}
			sequenceid = tx.SequenceID
			amount = casted.Amount

			opIterFunc, opCloseFunc := block.GetBlockOperationsBySource(api.storage, casted.Target, nil)
			state = resource.FrozenState
			for {
				bo, hasNext, _ := opIterFunc()
				switch bo.Type {
				case operation.TypeUnfreezingRequest:
					lastblock := block.GetLatestBlock(api.storage)
					if lastblock.Height-bo.Height >= common.UnfreezingPeriod {
						state = resource.UnfrozenState
					} else {
						unfreezingRemainingBlocks = bo.Height + common.UnfreezingPeriod - lastblock.Height
						state = resource.MeltingState
					}
					unfreezingOpHash = bo.OpHash
					unfreezingBlockHeight = bo.Height
				case operation.TypePayment:
					state = resource.ReturnedState
					paymentOpHash = bo.OpHash
				}
				if !hasNext {
					break
				}
			}
			opCloseFunc()

			info := resource.FrozenAccountInfo{
				CreatedBlockHeight:           createdBlockHeight,
				CreatedOpHash:                createdOpHash,
				CreatedSequenceId:            sequenceid,
				InitialAmount:                amount,
				FreezingState:                state,
				UnfreezingRequestBlockHeight: unfreezingBlockHeight,
				UnfreezingRequestOpHash:      unfreezingOpHash,
				UnfreezingRemainingBlocks:    unfreezingRemainingBlocks,
				PaymentOpHash:                paymentOpHash,
			}
			var ba *block.BlockAccount
			if ba, err = block.GetBlockAccount(api.storage, casted.Target); err != nil {
				break
			}

			frozenAccountResource := resource.NewFrozenAccount(ba, info)
			txs = append(txs, frozenAccountResource)
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("linked-%s", address)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		txs := readFunc()
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockOperationObserver, event)
		return
	}

	txs := readFunc()
	self := r.URL.String()
	next := strings.Replace(resource.URLAccountFrozenAccounts, "{id}", address, -1) + "?" + options.SetCursor(cursor).SetReverse(false).Encode()
	prev := strings.Replace(resource.URLAccountFrozenAccounts, "{id}", address, -1) + "?" + options.SetReverse(true).Encode()
	list := resource.NewResourceList(txs, self, next, prev)

	if err := httputils.WriteJSON(w, 200, list); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
}

func (api NetworkHandlerAPI) GetFrozenAccountsHandler(w http.ResponseWriter, r *http.Request) {
	options, err := storage.NewDefaultListOptionsFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, errors.InvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	var cursor []byte
	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockOperationsByFrozen(api.storage, options)
		for {
			var (
				createdBlockHeight        uint64
				createdOpHash             string
				sequenceid                uint64
				amount                    common.Amount
				state                     resource.FrozenAccountState
				unfreezingBlockHeight     uint64
				unfreezingOpHash          string
				unfreezingRemainingBlocks uint64
				paymentOpHash             string
			)

			bo, hasNext, c := iterFunc()
			cursor = c
			if !hasNext {
				break
			}
			var (
				casted operation.CreateAccount
				ok     bool
				body   operation.Body
			)
			if body, err = operation.UnmarshalBodyJSON(bo.Type, bo.Body); err != nil {
				break
			}
			if casted, ok = body.(operation.CreateAccount); !ok {
				break
			}
			createdBlockHeight = bo.Height
			createdOpHash = bo.OpHash
			var tx block.BlockTransaction
			if tx, err = block.GetBlockTransaction(api.storage, bo.TxHash); err != nil {
				break
			}
			sequenceid = tx.SequenceID
			amount = casted.Amount

			opIterFunc, opCloseFunc := block.GetBlockOperationsBySource(api.storage, casted.Target, nil)
			state = resource.FrozenState
			for {
				bo, hasNext, _ := opIterFunc()
				switch bo.Type {
				case operation.TypePayment:
					state = resource.ReturnedState
					paymentOpHash = bo.OpHash
				case operation.TypeUnfreezingRequest:
					lastblock := block.GetLatestBlock(api.storage)
					if lastblock.Height-bo.Height >= common.UnfreezingPeriod {
						state = resource.UnfrozenState
					} else {
						unfreezingRemainingBlocks = bo.Height + uint64(241920) - lastblock.Height
						state = resource.MeltingState
					}
					unfreezingOpHash = bo.OpHash
					unfreezingBlockHeight = bo.Height
				}
				if !hasNext {
					break
				}
			}
			opCloseFunc()

			info := resource.FrozenAccountInfo{
				CreatedBlockHeight:           createdBlockHeight,
				CreatedOpHash:                createdOpHash,
				CreatedSequenceId:            sequenceid,
				InitialAmount:                amount,
				FreezingState:                state,
				UnfreezingRequestBlockHeight: unfreezingBlockHeight,
				UnfreezingRequestOpHash:      unfreezingOpHash,
				UnfreezingRemainingBlocks:    unfreezingRemainingBlocks,
				PaymentOpHash:                paymentOpHash,
			}
			var ba *block.BlockAccount
			if ba, err = block.GetBlockAccount(api.storage, casted.Target); err != nil {
				break
			}

			frozenAccountResource := resource.NewFrozenAccount(ba, info)
			txs = append(txs, frozenAccountResource)
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := "frozen"
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		txs := readFunc()
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockOperationObserver, event)
		return
	}

	txs := readFunc()
	self := r.URL.String()
	next := GetFrozenAccountHandlerPattern + "?" + options.SetCursor(cursor).SetReverse(false).Encode()
	prev := GetFrozenAccountHandlerPattern + "?" + options.SetReverse(true).Encode()
	list := resource.NewResourceList(txs, self, next, prev)

	if err := httputils.WriteJSON(w, 200, list); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
}
