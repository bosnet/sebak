package sebak

import (
	"github.com/nvellon/hal"
	"strings"
)

type APIResource interface {
	LinkSelf() string
	Resource(selfUrl string) *hal.Resource
	GetMap() hal.Entry
}

type APIResourceList []APIResource

func (al APIResourceList) Resource(selfUrl string) *hal.Resource {
	rl := hal.NewResource(struct{}{}, selfUrl)
	for _, apiResource := range al {
		r := apiResource.Resource(apiResource.LinkSelf())
		rl.Embed("records", r)
	}
	rl.AddLink("prev", hal.NewLink(selfUrl)) //TODO: set prev/next url
	rl.AddLink("next", hal.NewLink(selfUrl))

	return rl
}

func (al APIResourceList) LinkSelf() string {
	return ""
}
func (al APIResourceList) GetMap() hal.Entry {
	return hal.Entry{}
}

const (
	UrlAccounts     = "/accounts/{id}"
	UrlTransactions = "/transactions/{id}"
	UrlOperations   = "/operations/{id}"
)

type APIResourceAccount struct {
	accountId  string
	checkpoint string
	balance    string
}

func (aa APIResourceAccount) GetMap() hal.Entry {
	return hal.Entry{
		"id":         aa.accountId,
		"account_id": aa.accountId,
		"checkpoint": aa.checkpoint,
		"balance":    aa.balance,
	}
}

func (aa APIResourceAccount) Resource(selfUrl string) *hal.Resource {
	r := hal.NewResource(aa, selfUrl)
	r.AddLink("transactions", hal.NewLink(strings.Replace(UrlAccounts, "{id}", aa.accountId, -1)+"/transactions{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	r.AddLink("operations", hal.NewLink(strings.Replace(UrlAccounts, "{id}", aa.accountId, -1)+"/operations{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	return r
}

func (aa APIResourceAccount) LinkSelf() string {
	return strings.Replace(UrlAccounts, "{id}", aa.accountId, -1)
}

type APIResourceTransaction struct {
	hash               string
	previousCheckpoint string
	sourceCheckpoint   string
	targetCheckpoint   string
	signature          string
	source             string
	fee                string
	amount             string
	created            string
	operations         []string
}

func (at APIResourceTransaction) GetMap() hal.Entry {
	return hal.Entry{
		"id":                at.hash,
		"hash":              at.hash,
		"account":           at.source,
		"fee_paid":          at.fee,
		"source_checkpoint": at.sourceCheckpoint,
		"target_checkpoint": at.targetCheckpoint,
		"created_at":        at.created,
		"operation_count":   len(at.operations),
	}
}
func (at APIResourceTransaction) Resource(selfUrl string) *hal.Resource {

	r := hal.NewResource(at, selfUrl)
	r.AddLink("accounts", hal.NewLink(strings.Replace(UrlAccounts, "{id}", at.source, -1)))
	r.AddLink("operations", hal.NewLink(strings.Replace(UrlTransactions, "{id}", at.hash, -1)+"/operations{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	return r
}

func (at APIResourceTransaction) LinkSelf() string {
	return strings.Replace(UrlTransactions, "{id}", at.hash, -1)
}

type APIResourceOperation struct {
	hash    string
	txHash  string
	funder  string //Source Account
	account string //Target Account
	otype   string
	amount  string
}

func (ao APIResourceOperation) GetMap() hal.Entry {
	return hal.Entry{
		"id":      ao.hash,
		"hash":    ao.hash,
		"funder":  ao.funder,
		"account": ao.account,
		"type":    ao.otype,
		"amount":  ao.amount,
	}
}

func (ao APIResourceOperation) Resource(selfUrl string) *hal.Resource {

	r := hal.NewResource(ao, selfUrl)
	r.AddNewLink("transactions", strings.Replace(UrlTransactions, "{id}", ao.txHash, -1))
	return r
}

func (ao APIResourceOperation) LinkSelf() string {
	return strings.Replace(UrlOperations, "{id}", ao.hash, -1)
}
