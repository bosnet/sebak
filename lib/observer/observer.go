package observer

import "github.com/GianlucaGuarini/go-observable"

var BlockAccountObserver = observable.New()
var BlockTransactionObserver = observable.New()
var BlockOperationObserver = observable.New()
var NodeObserver = observable.New()
