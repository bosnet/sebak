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
	return hal.Entry{"a": "b"}
}

const (
	UrlAccounts     = "/accounts/{id}"
	UrlTransactions = "/transactions/{id}"
	UrlOperations   = "/operations/{id}"
)

type APIResourceAccount struct {
	id         string
	accountId  string
	checkpoint string
	balance    string
}

func (aa APIResourceAccount) GetMap() hal.Entry {
	return hal.Entry{
		"id":         aa.id,
		"account_id": aa.accountId,
		"checkpoint": aa.checkpoint,
		"balance":    aa.balance,
	}
}

func (aa APIResourceAccount) Resource(selfUrl string) *hal.Resource {
	r := hal.NewResource(aa, selfUrl)
	r.AddNewLink("transactions", strings.Replace(UrlAccounts, "{id}", aa.id, -1)+"/transactions{?cursor,limit,order}")
	r.AddNewLink("operations", strings.Replace(UrlAccounts, "{id}", aa.id, -1)+"/operations{?cursor,limit,order}")
	return r
}

func (aa APIResourceAccount) LinkSelf() string {
	return strings.Replace(UrlAccounts, "{id}", aa.id, -1)
}

type APIResourceTransaction struct {
	id               string
	hash             string
	account          string //Source Account
	feePaid          string
	sourceCheckpoint string
	targetCheckpoint string
	createdAt        string //confirmed? created?
	operationCount   uint64
}

func (at APIResourceTransaction) GetMap() hal.Entry {
	return hal.Entry{
		"id":               at.id,
		"hash":             at.hash,
		"account":          at.account,
		"fee_paid":         at.feePaid,
		"sourceCheckpoint": at.sourceCheckpoint,
		"targetCheckpoint": at.targetCheckpoint,
		"created_at":       at.createdAt,
		"operationCount":   at.operationCount,
	}
}
func (at APIResourceTransaction) Resource(selfUrl string) *hal.Resource {

	r := hal.NewResource(at, selfUrl)
	r.AddNewLink("accounts", strings.Replace(UrlAccounts, "{id}", at.account, -1))
	r.AddNewLink("operations", strings.Replace(UrlTransactions, "{id}", at.id, -1)+"/operations{?cursor,limit,order}")
	return r
}

func (at APIResourceTransaction) LinkSelf() string {
	return strings.Replace(UrlTransactions, "{id}", at.id, -1)
}

type APIResourceOperation struct {
	id      string
	hash    string
	funder  string //Source Account
	account string //Target Account
	otype   string
	amount  string
}

func (ao APIResourceOperation) GetMap() hal.Entry {
	return hal.Entry{
		"id":      ao.id,
		"hash":    ao.hash,
		"funder":  ao.funder,
		"account": ao.account,
		"type":    ao.otype,
		"amount":  ao.amount,
	}
}

func (ao APIResourceOperation) Resource(selfUrl string) *hal.Resource {

	r := hal.NewResource(ao, selfUrl)
	r.AddNewLink("transactions", strings.Replace(UrlTransactions, "{id}", ao.id, -1))
	return r
}

func (ao APIResourceOperation) LinkSelf() string {
	return strings.Replace(UrlOperations, "{id}", ao.id, -1)
}
