package observer

import (
	"github.com/GianlucaGuarini/go-observable"
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
	ConditionAll            = "*"
	ConditionSource         = "source"
	ConditionTarget         = "target"
	ConditionType           = "type"
	ConditionOpHash         = "ophash"
	ConditionTxHash         = "txhash"
	ConditionAddress        = "address"
)

type Event struct {
	Resource  string `json:"resource"`
	Condition string `json:"condition"`
	Id        string `json:"id"`
}

func NewEvent(resource, condition, id string) Event {
	return Event{
		Resource:  resource,
		Condition: condition,
		Id:        id,
	}
}
func (e Event) String() string {
	toStr := e.Resource + "-"
	if e.Condition == ConditionAll {
		toStr += e.Condition
	} else {
		toStr += e.Condition + "="
		toStr += e.Id
	}
	return toStr
}

type Subscribe struct {
	Events []Event `json:"resources"`
}

func NewSubscribe(events ...Event) Subscribe {
	s := Subscribe{}
	for _, e := range events {
		s.Events = append(s.Events, e)
	}
	return s
}

func (s Subscribe) String() string {
	toStr := ""
	for i, e := range s.Events {
		toStr += e.String()
		if i == len(s.Events)-1 && len(s.Events) != 1 {
			toStr += "&"
		}
	}
	return toStr
}
