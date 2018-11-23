package observer

import (
	"github.com/GianlucaGuarini/go-observable"
	"strings"
)

var SyncBlockWaitObserver = observable.New()

var ResourceObserver = observable.New()

type ResourceTy = string

type KeyTy = string

const (
	Tx     ResourceTy = "tx"
	TxPool            = "txpool"
	Op                = "op"
	Acc               = "acc"
)

const (
	All     KeyTy = "*"
	Source        = "source"
	Target        = "target"
	Type          = "type"
	OpHash        = "ophash"
	TxHash        = "txhash"
	Address       = "address"
)

type Condition struct {
	Resource ResourceTy `json:"resource"`
	Key      KeyTy      `json:"key"`
	Value    string     `json:"value"`
}

func NewCondition(resource ResourceTy, key KeyTy, v ...string) Condition {
	value := ""
	if len(v) > 0 {
		value = v[0]
	}
	return Condition{
		Resource: resource,
		Key:      key,
		Value:    value,
	}
}

func (c Condition) Event() string {
	toStr := c.Resource + "-"
	if c.Key == All {
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

func Event(conditions ...Condition) string {
	return (Conditions)(conditions).Event()
}
