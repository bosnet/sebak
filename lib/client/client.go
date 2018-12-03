package client

import (
	"boscoin.io/sebak/lib/common/observer"
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"sync"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
)

const (
	UrlPrefixForAPIV1 = "/api/v1"

	UrlAccountTransactions   = "/accounts/{id}/transactions"
	UrlAccount               = "/accounts/{id}"
	UrlAccountOperations     = "/accounts/{id}/operations"
	UrlAccountFrozenAccounts = "/accounts/{id}/frozen-accounts"
	UrlFrozenAccounts        = "/frozen-accounts"
	UrlTransactions          = "/transactions"
	UrlTransactionByHash     = "/transactions/{id}"
	UrlTransactionStatus     = "/transactions/{id}/status"
	UrlTransactionOperations = "/transactions/{id}/operations"
	UrlSubscribe             = "/subscribe"
)

type QueryKey string

func (qk QueryKey) String() string {
	return string(qk)
}

const (
	QueryLimit  QueryKey = "limit"
	QueryOrder  QueryKey = "reverse"
	QueryCursor QueryKey = "cursor"
	QueryType   QueryKey = "type"
)

type Q struct {
	Key   QueryKey
	Value string
}

type Queries []Q

func (qs Queries) toQueryString() string {
	urlValues := neturl.Values{}
	if len(qs) == 0 {
		return ""
	}
	for _, q := range qs {
		switch q.Key {
		case QueryLimit:
			urlValues.Add(QueryLimit.String(), q.Value)
		case QueryOrder:
			urlValues.Add(QueryOrder.String(), q.Value)
		case QueryCursor:
			urlValues.Add(QueryCursor.String(), q.Value)
		case QueryType:
			urlValues.Add(QueryType.String(), q.Value)

		}
	}
	return "?" + urlValues.Encode()
}

type Client struct {
	URL string

	HTTP *common.HTTP2Client
}

func NewClient(url string) *Client {
	httpClient, err := common.NewHTTP2Client(0, 0, true)
	if err != nil {
		panic(err)
	}
	return &Client{
		URL:  url,
		HTTP: httpClient,
	}
}

// NewDefaultListOptionsFromQuery makes ListOptions from url.Query.
func NewDefaultListOptionsFromQuery(v neturl.Values) (options *storage.DefaultListOptions, err error) {
	var reverse bool
	var cursor []byte
	var limit uint64 = storage.DefaultMaxLimitListOptions

	r := v.Get(string(QueryOrder))
	if len(r) > 0 {
		if reverse, err = common.ParseBoolQueryString(r); err != nil {
			return nil, err
		}
	}

	r = v.Get(string(QueryCursor))
	if len(r) > 0 {
		cursor = []byte(r)
	}

	r = v.Get(string(QueryLimit))
	if len(r) > 0 {
		if limit, err = strconv.ParseUint(r, 10, 64); err != nil {
			return nil, err
		}
	}

	return storage.NewDefaultListOptions(reverse, cursor, limit), nil
}

func (c *Client) ToResponse(resp *http.Response, response interface{}) (err error) {
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	if !(resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices) {
		var p Problem
		err = decoder.Decode(&p)
		if err != nil {
			return
		}
		return Error{Problem: p}
	}

	err = decoder.Decode(&response)
	if err != nil {
		return
	}
	return
}

func (c *Client) Get(path string, headers http.Header) (response *http.Response, err error) {

	url := c.URL + UrlPrefixForAPIV1 + path
	return c.HTTP.Get(url, headers)
}

func (c *Client) getResponse(url string, headers http.Header, response interface{}) (err error) {
	if len(headers.Get("Content-Type")) == 0 {
		headers.Set("Content-Type", "application/json")
	}
	resp, err := c.Get(url, headers)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	return c.ToResponse(resp, &response)
}

func (c *Client) Post(path string, body []byte, headers http.Header) (response *http.Response, err error) {
	url := c.URL + UrlPrefixForAPIV1 + path
	return c.HTTP.Post(url, body, headers)
}

func (c *Client) LoadAccount(id string, queries ...Q) (account Account, err error) {
	url := strings.Replace(UrlAccount, "{id}", id, -1)
	url += Queries(queries).toQueryString()
	err = c.getResponse(url, http.Header{}, &account)
	return
}

func (c *Client) LoadFrozenAccountsByLinked(id string, queries ...Q) (fPage FrozenAccountsPage, err error) {
	url := strings.Replace(UrlAccountFrozenAccounts, "{id}", id, -1)
	url += Queries(queries).toQueryString()
	err = c.getResponse(url, http.Header{}, &fPage)
	return
}

func (c *Client) LoadAFrozenAccounts(id string, queries ...Q) (fPage FrozenAccountsPage, err error) {
	url := strings.Replace(UrlFrozenAccounts, "{id}", id, -1)
	url += Queries(queries).toQueryString()
	err = c.getResponse(url, http.Header{}, &fPage)
	return
}

func (c *Client) LoadTransaction(id string, queries ...Q) (transaction Transaction, err error) {
	url := strings.Replace(UrlTransactionByHash, "{id}", id, -1)
	url += Queries(queries).toQueryString()
	err = c.getResponse(url, http.Header{}, &transaction)
	return
}

func (c *Client) LoadTransactionStatus(id string, queries ...Q) (transactionHistory TransactionStatus, err error) {
	url := strings.Replace(UrlTransactionStatus, "{id}", id, -1)
	url += Queries(queries).toQueryString()
	err = c.getResponse(url, http.Header{}, &transactionHistory)
	return
}

func (c *Client) LoadTransactions(queries ...Q) (tPage TransactionsPage, err error) {
	url := UrlTransactions
	url += Queries(queries).toQueryString()
	err = c.getResponse(url, http.Header{}, &tPage)
	return
}

func (c *Client) LoadTransactionsByAccount(id string, queries ...Q) (tPage TransactionsPage, err error) {
	url := strings.Replace(UrlAccountTransactions, "{id}", id, -1)
	url += Queries(queries).toQueryString()
	err = c.getResponse(url, http.Header{}, &tPage)
	return
}

func (c *Client) LoadOperationsByAccount(id string, queries ...Q) (oPage OperationsPage, err error) {
	url := strings.Replace(UrlAccountOperations, "{id}", id, -1)
	url += Queries(queries).toQueryString()
	err = c.getResponse(url, http.Header{}, &oPage)
	return
}

func (c *Client) LoadOperationsByTransaction(id string, queries ...Q) (oPage OperationsPage, err error) {
	url := strings.Replace(UrlTransactionOperations, "{id}", id, -1)
	url += Queries(queries).toQueryString()
	err = c.getResponse(url, http.Header{}, &oPage)
	return
}

func (c *Client) SubmitTransaction(tx []byte) (pTransaction TransactionPost, err error) {
	url := UrlTransactions
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := c.Post(url, tx, headers)
	defer resp.Body.Close()
	if err != nil {
		return
	}
	err = c.ToResponse(resp, &pTransaction)
	return
}

func (c *Client) SubmitTransactionAndWait(hash string, tx []byte) (pTransaction TransactionPost, err error) {
	var wg sync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err = c.StreamTransactionStatus(ctx, hash, nil, func(status TransactionStatus) {
			if status.Status == "confirmed" {
				pTransaction.Status = status.Status
				cancel()
			}
		})
		wg.Done()
	}()

	pTransaction, err = c.SubmitTransaction(tx)
	if err != nil {
		cancel()
		return pTransaction, err
	}

	wg.Wait()

	return
}

func (c *Client) Stream(ctx context.Context, url string, body []byte, handler func(data []byte) error) (err error) {
	var headers = http.Header{}
	headers.Set("Accept", "text/event-stream")
	var resp *http.Response
	if body != nil {
		resp, err = c.Post(url, body, headers)
	} else {
		resp, err = c.Get(url, headers)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	readChan := make(chan []byte)
	errChan := make(chan error)

	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				errChan <- err
				break
			}
			readChan <- line
		}
	}()

	for {
		breakFor := false
		select {
		case <-ctx.Done():
			resp.Body.Close()
			breakFor = true
		case err = <-errChan:
			breakFor = true
		case line := <-readChan:
			if len(line) == 0 {
				continue
			}
			handler(line)
		}
		if breakFor {
			break
		}
	}

	return
}

func (c *Client) StreamAccount(ctx context.Context, id string, handler func(Account)) (err error) {
	s := []observer.Conditions{{observer.NewCondition(observer.Acc, observer.All)}}
	b, err := json.Marshal(s)
	handlerFunc := func(b []byte) (err error) {
		var v Account
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, UrlSubscribe, b, handlerFunc)
}

func (c *Client) StreamTransactions(ctx context.Context, handler func(Transaction)) (err error) {
	s := []observer.Conditions{{observer.NewCondition(observer.Tx, observer.All)}}
	b, err := json.Marshal(s)
	handlerFunc := func(b []byte) (err error) {
		var v Transaction
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, UrlSubscribe, b, handlerFunc)
}

func (c *Client) StreamTransactionsByAccount(ctx context.Context, id string, handler func(Transaction)) (err error) {
	s := []observer.Conditions{{observer.NewCondition(observer.Tx, observer.Source, id), observer.NewCondition(observer.Tx, observer.Target, id)}}
	b, err := json.Marshal(s)
	handlerFunc := func(b []byte) (err error) {
		var v Transaction
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, UrlSubscribe, b, handlerFunc)
}

func (c *Client) StreamTransactionStatus(ctx context.Context, id string, body []byte, handler func(TransactionStatus)) (err error) {
	url := strings.Replace(UrlTransactionStatus, "{id}", id, -1)
	handlerFunc := func(b []byte) (err error) {
		var v TransactionStatus
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, url, nil, handlerFunc)
}

func (c *Client) StreamTransactionsByHash(ctx context.Context, id string, handler func(Transaction)) (err error) {
	s := []observer.Conditions{{observer.NewCondition(observer.Tx, observer.TxHash, id)}}
	b, err := json.Marshal(s)
	handlerFunc := func(b []byte) (err error) {
		var v Transaction
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, UrlSubscribe, b, handlerFunc)
}
