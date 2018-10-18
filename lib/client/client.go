package client

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	neturl "net/url"
	"strings"

	"boscoin.io/sebak/lib/common"
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
	UrlTransactionHistory    = "/transactions/{id}/history"
	UrlTransactionOperations = "/transactions/{id}/operations"
)

type QueryKey string

func (qk QueryKey) String() string {
	return string(qk)
}

const (
	QueryLimit  QueryKey = "limit"
	QueryOrder  QueryKey = "order"
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

func (c *Client) LoadTransactionHistory(id string, queries ...Q) (transactionHistory TransactionHistory, err error) {
	url := strings.Replace(UrlTransactionHistory, "{id}", id, -1)
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

func (c *Client) Stream(ctx context.Context, theUrl string, cursor *string, handler func(data []byte) error) (err error) {
	query := neturl.Values{}
	if cursor != nil {
		query.Set("cursor", string(*cursor))
	}
	theUrl += "?" + query.Encode()
	var headers = http.Header{}
	headers.Set("Accept", "text/event-stream")
	resp, err := c.Get(theUrl, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for true {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}

		if len(line) == 0 {
			continue
		}
		handler(line)

		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}

	return
}

func (c *Client) StreamAccount(ctx context.Context, id string, cursor *string, handler func(Account)) (err error) {
	url := strings.Replace(UrlAccount, "{id}", id, -1)
	handlerFunc := func(b []byte) (err error) {
		var v Account
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, url, cursor, handlerFunc)
}

func (c *Client) StreamFrozenAccountsByLinked(ctx context.Context, id string, cursor *string, handler func(FrozenAccount)) (err error) {
	url := strings.Replace(UrlAccountFrozenAccounts, "{id}", id, -1)
	handlerFunc := func(b []byte) (err error) {
		var v FrozenAccount
		err = json.Unmarshal(b, &v)
		if err != nil {
			return
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, url, cursor, handlerFunc)
}

func (c *Client) StreamFrozenAccounts(ctx context.Context, id string, cursor *string, handler func(FrozenAccount)) (err error) {
	url := strings.Replace(UrlFrozenAccounts, "{id}", id, -1)
	handlerFunc := func(b []byte) (err error) {
		var v FrozenAccount
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, url, cursor, handlerFunc)
}

func (c *Client) StreamTransactions(ctx context.Context, cursor *string, handler func(Transaction)) (err error) {
	url := UrlTransactions
	handlerFunc := func(b []byte) (err error) {
		var v Transaction
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, url, cursor, handlerFunc)
}

func (c *Client) StreamTransactionsByAccount(ctx context.Context, id string, cursor *string, handler func(Transaction)) (err error) {
	url := strings.Replace(UrlAccountTransactions, "{id}", id, -1)
	handlerFunc := func(b []byte) (err error) {
		var v Transaction
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, url, cursor, handlerFunc)
}

func (c *Client) StreamTransactionsByHash(ctx context.Context, id string, cursor *string, handler func(Transaction)) (err error) {
	url := strings.Replace(UrlTransactionByHash, "{id}", id, -1)
	handlerFunc := func(b []byte) (err error) {
		var v Transaction
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, url, cursor, handlerFunc)
}

func (c *Client) StreamOperationsByAccount(ctx context.Context, id string, cursor *string, handler func(Operation)) (err error) {
	url := strings.Replace(UrlAccountOperations, "{id}", id, -1)
	handlerFunc := func(b []byte) (err error) {
		var v Operation
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, url, cursor, handlerFunc)
}

func (c *Client) StreamOperationsByTransaction(ctx context.Context, id string, cursor *string, handler func(Operation)) (err error) {
	url := strings.Replace(UrlTransactionOperations, "{id}", id, -1)
	handlerFunc := func(b []byte) (err error) {
		var v Operation
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}
		handler(v)
		return nil
	}
	return c.Stream(ctx, url, cursor, handlerFunc)
}
