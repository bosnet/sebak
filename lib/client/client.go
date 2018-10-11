package client

import (
	"boscoin.io/sebak/lib/common"
	"bufio"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	UrlPrefixForAPIV1 = "/api/v1"

	UrlAccountTransactions   = "/accounts/{id}/transactions"
	UrlAccount               = "/accounts/{id}"
	UrlAccountOperations     = "/accounts/{id}/operations"
	UrlTransactions          = "/transactions"
	UrlTransactionByHash     = "/transactions/{id}"
	UrlTransactionOperations = "/transactions/{id}/operations"
)

type QueryParams struct {
	Limit  int
	Order  bool
	Cursor string
}

func (q QueryParams) toUrlValues() url.Values {
	query := url.Values{}
	if q.Limit > 0 {
		query.Add("limit", strconv.Itoa(q.Limit))
	}

	if q.Order {
		query.Add("order", "true")
	} else {
		query.Add("order", "false")
	}

	if len(q.Cursor) > 0 {
		query.Add("cursor", q.Cursor)
	}

	return query
}

type Client struct {
	URL string

	HTTP *common.HTTP2Client
}

func NewClient(url string) *Client {
	httpClient, err := common.NewHTTP2Client(0, 0, true)
	if err != nil {
		//TODO: handle Error
	}
	return &Client{
		URL:  url,
		HTTP: httpClient,
	}
}

func (c *Client) toResponse(resp *http.Response, response interface{}) (err error) {
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

func (c *Client) Post(path string, body []byte, headers http.Header) (response *http.Response, err error) {
	url := c.URL + UrlPrefixForAPIV1 + path
	return c.HTTP.Post(url, body, headers)
}

func (c *Client) LoadAccount(id string) (account Account, err error) {
	url := strings.Replace(UrlAccount, "{id}", id, -1)
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := c.Get(url, headers)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.toResponse(resp, &account)
	return
}

func (c *Client) LoadTransaction(id string) (transaction Transaction, err error) {
	url := strings.Replace(UrlTransactionByHash, "{id}", id, -1)
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := c.Get(url, headers)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.toResponse(resp, &transaction)
	return
}

func (c *Client) LoadTransactions() (tPage TransactionsPage, err error) {
	url := UrlTransactions
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := c.Get(url, headers)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.toResponse(resp, &tPage)
	return
}

func (c *Client) LoadTransactionsByAccount(id string) (tPage TransactionsPage, err error) {
	url := strings.Replace(UrlAccountTransactions, "{id}", id, -1)
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := c.Get(url, headers)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.toResponse(resp, &tPage)
	return
}

func (c *Client) LoadOperationsByAccount(id string) (oPage OperationsPage, err error) {
	url := strings.Replace(UrlAccountOperations, "{id}", id, -1)
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := c.Get(url, headers)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.toResponse(resp, &oPage)
	return
}

func (c *Client) LoadOperationsByTransaction(id string) (oPage OperationsPage, err error) {
	url := strings.Replace(UrlTransactionOperations, "{id}", id, -1)
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := c.Get(url, headers)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.toResponse(resp, &oPage)
	return
}

func (c *Client) SubmitTransaction(tx []byte) (retBody []byte, err error) { //TODO: make a model for the retBody
	url := UrlTransactions
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp, err := c.Post(url, tx, headers)
	if resp.StatusCode == http.StatusOK {
		retBody, err = ioutil.ReadAll(resp.Body)
	}
	defer resp.Body.Close()
	return
}

func (c *Client) Stream(ctx context.Context, theUrl string, cursor *string, handler func(data []byte) error) (err error) {
	query := url.Values{}
	if cursor != nil {
		query.Set("cursor", string(*cursor))
	}
	theUrl += "?" + query.Encode()
	var headers = http.Header{}
	headers.Set("Accept", "text/event-stream")
	resp, err := c.HTTP.Get(theUrl, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if len(scanner.Bytes()) == 0 {
			continue
		}
		data := scanner.Bytes()
		handler(data)
	}

	err = scanner.Err()

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
