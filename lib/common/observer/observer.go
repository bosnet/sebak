package observer

import "github.com/GianlucaGuarini/go-observable"

var BlockAccountObserver = observable.New()
var BlockTransactionObserver = observable.New()
var BlockTransactionHistoryObserver = observable.New()
var BlockObserver = observable.New()
var BlockOperationObserver = observable.New()
var SyncBlockWaitObserver = observable.New()
