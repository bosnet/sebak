package observer

import (
	"github.com/GianlucaGuarini/go-observable"
	"strings"
)

var BlockAccountObserver = observable.New()
var BlockTransactionObserver = observable.New()
var BlockObserver = observable.New()
var BlockOperationObserver = observable.New()
var SyncBlockWaitObserver = observable.New()

var ResourceObserver = observable.New()

const (
	ResourceTransaction     = "tx"
	ResourceTransactionPool = "txpool"
	ResourceOperation       = "op"
	ResourceAccount         = "ac"
	KeyAll                  = "*"
	KeySource               = "source"
	KeyTarget               = "target"
	KeyType                 = "type"
	KeyOpHash               = "ophash"
	KeyTxHash               = "txhash"
	KeyAddress              = "address"
)

type Event interface {
	Event() string
}

type Condition struct {
	Resource string `json:"resource"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

func NewCondition(resource, key, value string) Condition {
	return Condition{
		Resource: resource,
		Key:      key,
		Value:    value,
	}
}

func (c Condition) Event() string {
	toStr := c.Resource + "-"
	if c.Key == KeyAll {
		toStr += c.Key
	} else {
		toStr += c.Key + "="
		toStr += c.Value
	}
	return toStr
}

type Conditions []Condition

func (cs Conditions) Event() string {
	var ss []string
	for _, c := range cs {
		ss = append(ss, c.Event())
	}
	return strings.Join(ss, "&")
}
