package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
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

	payload, err := readFunc()
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	httputils.MustWriteJSON(w, 200, payload)
}

func (api NetworkHandlerAPI) GetAccountsHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	var addresses []string
	if err := json.Unmarshal(body, &addresses); err != nil {
		httputils.WriteJSONError(w, errors.BadRequestParameter.Clone().SetData("error", err.Error()))
		return
	}
	if uint64(len(addresses)) > DefaultLimit {
		httputils.WriteJSONError(w, errors.PageQueryLimitMaxExceed)
		return
	} else if len(addresses) < 1 {
		httputils.WriteJSONError(w, errors.BadRequestParameter)
		return
	}

	var rs []resource.Resource
	for _, address := range addresses {
		found, err := block.ExistsBlockAccount(api.storage, address)
		if err != nil {
			httputils.WriteJSONError(w, err)
			return
		}
		if !found {
			continue
		}
		ba, err := block.GetBlockAccount(api.storage, address)
		if err != nil {
			httputils.WriteJSONError(w, err)
			return
		}
		rs = append(rs, resource.NewAccount(ba))
	}

	httputils.MustWriteJSON(w, 200, resource.NewResourceList(rs, "", "", ""))
}

func (api NetworkHandlerAPI) GetFrozenAccountsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]

	p, err := NewPageQuery(r)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	var options = p.ListOptions()
	var firstCursor []byte
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
			if !hasNext {
				break
			}
			cursor = append([]byte{}, c...)
			if len(firstCursor) == 0 {
				firstCursor = append(firstCursor, c...)
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

	txs := readFunc()

	list := p.ResourceList(txs, firstCursor, cursor)
	httputils.MustWriteJSON(w, 200, list)
}

func (api NetworkHandlerAPI) GetFrozenAccountsHandler(w http.ResponseWriter, r *http.Request) {
	p, err := NewPageQuery(r)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	var options = p.ListOptions()
	var firstCursor []byte
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
			if !hasNext {
				break
			}
			cursor = append([]byte{}, c...)
			if len(firstCursor) == 0 {
				firstCursor = append(firstCursor, c...)
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
						unfreezingRemainingBlocks = bo.Height + common.UnfreezingPeriod - lastblock.Height
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

	txs := readFunc()
	list := p.ResourceList(txs, firstCursor, cursor)
	httputils.MustWriteJSON(w, 200, list)
}
