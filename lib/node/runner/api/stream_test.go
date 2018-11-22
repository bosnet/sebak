package api

import (
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/transaction"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"boscoin.io/sebak/lib/block"
	"github.com/GianlucaGuarini/go-observable"

	"github.com/stretchr/testify/require"
)

func TestAccountStream(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()
	kp, _, bt, bo := prepareBlkTxOpWithoutSave(storage)
	ba := block.TestMakeBlockAccount(kp)

	// Save
	{
		ba.MustSave(storage)
		bt.MustSave(storage)
		bo.MustSave(storage)
	}

	// Do a Request
	var acReader *bufio.Reader
	var txReader *bufio.Reader
	var opReader *bufio.Reader
	{
		s := []observer.Subscribe{observer.NewSubscribe(observer.NewEvent(observer.ResourceAccount, observer.ConditionAddress, ba.Address))}
		b, err := json.Marshal(s)
		require.NoError(t, err)
		respBody := request(ts, PostSubscribePattern, true, b)
		defer respBody.Close()
		acReader = bufio.NewReader(respBody)
	}
	{
		s := []observer.Subscribe{observer.NewSubscribe(observer.NewEvent(observer.ResourceTransaction, observer.ConditionTxHash, bt.Hash))}
		b, err := json.Marshal(s)
		require.NoError(t, err)
		respBody := request(ts, PostSubscribePattern, true, b)
		defer respBody.Close()
		txReader = bufio.NewReader(respBody)
	}
	{
		s := []observer.Subscribe{observer.NewSubscribe(observer.NewEvent(observer.ResourceOperation, observer.ConditionOpHash, bo.Hash))}
		b, err := json.Marshal(s)
		require.NoError(t, err)
		respBody := request(ts, PostSubscribePattern, true, b)
		defer respBody.Close()
		opReader = bufio.NewReader(respBody)
	}
	tx := bt.Transaction()
	txs := []*transaction.Transaction{&tx}
	TriggerEvent(storage, txs)

	// Check the output
	{
		line, err := acReader.ReadBytes('\n')
		require.NoError(t, err)
		line = bytes.Trim(line, "\n")
		if len(line) == 0 {
			line, err = acReader.ReadBytes('\n')
			require.NoError(t, err)
			line = bytes.Trim(line, "\n")
		}
		recv := make(map[string]interface{})
		json.Unmarshal(line, &recv)
		require.Equal(t, ba.Address, recv["address"], "address is not same")
	}
	{
		line, err := txReader.ReadBytes('\n')
		require.NoError(t, err)
		line = bytes.Trim(line, "\n")
		if len(line) == 0 {
			line, err = txReader.ReadBytes('\n')
			require.NoError(t, err)
			line = bytes.Trim(line, "\n")
		}
		recv := make(map[string]interface{})
		json.Unmarshal(line, &recv)
		require.Equal(t, bt.Hash, recv["hash"], "hash is not the same")
		require.Equal(t, bt.Block, recv["block"], "block is not the same")
	}
	{
		line, err := opReader.ReadBytes('\n')
		require.NoError(t, err)
		line = bytes.Trim(line, "\n")
		if len(line) == 0 {
			line, err = opReader.ReadBytes('\n')
			require.NoError(t, err)
			line = bytes.Trim(line, "\n")
		}
		recv := make(map[string]interface{})
		json.Unmarshal(line, &recv)
		require.Equal(t, bo.Hash, recv["hash"], "hash is not same")
	}

}

func TestAPIStreamRun(t *testing.T) {
	tests := []struct {
		name       string
		events     []string
		makeStream func(http.ResponseWriter, *http.Request) *EventStream
		trigger    func(*observable.Observable)
		respFunc   func(testing.TB, *http.Response)
	}{
		{
			"default",
			[]string{"test1"},
			func(w http.ResponseWriter, r *http.Request) *EventStream {
				es := NewDefaultEventStream(w, r)
				return es
			},
			func(ob *observable.Observable) {
				ob.Trigger("test1", block.NewBlockAccount("hello", 100))
			},
			func(t testing.TB, res *http.Response) {
				s := bufio.NewScanner(res.Body)
				s.Scan()

				var ba block.BlockAccount
				require.Nil(t, json.Unmarshal(s.Bytes(), &ba))
				require.Nil(t, s.Err())
				require.Equal(t, ba, *block.NewBlockAccount("hello", 100))
			},
		},
		{
			"renderFunc",
			[]string{"test1"},
			func(w http.ResponseWriter, r *http.Request) *EventStream {
				renderFunc := func(args ...interface{}) ([]byte, error) {
					s, ok := args[1].(*block.BlockAccount)
					if !ok {
						return nil, fmt.Errorf("this is not serializable")
					}
					bs, err := s.Serialize()
					if err != nil {
						return nil, err
					}
					return bs, nil
				}
				es := NewEventStream(w, r, renderFunc, DefaultContentType)
				return es
			},
			func(ob *observable.Observable) {
				ob.Trigger("test1", block.NewBlockAccount("hello", 100))
			},
			func(t testing.TB, res *http.Response) {
				s := bufio.NewScanner(res.Body)
				s.Scan()

				var ba block.BlockAccount
				require.Nil(t, json.Unmarshal(s.Bytes(), &ba))
				require.Nil(t, s.Err())
				require.Equal(t, ba, *block.NewBlockAccount("hello", 100))
			},
		},
		{
			"renderBeforeObservable",
			[]string{"test1"},
			func(w http.ResponseWriter, r *http.Request) *EventStream {
				es := NewDefaultEventStream(w, r)
				es.Render(block.NewBlockAccount("hello", 100))
				return es
			},
			nil, // no trigger
			func(t testing.TB, res *http.Response) {
				s := bufio.NewScanner(res.Body)
				s.Scan()

				var ba block.BlockAccount
				require.Nil(t, json.Unmarshal(s.Bytes(), &ba))
				require.Nil(t, s.Err())
				require.Equal(t, ba, *block.NewBlockAccount("hello", 100))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ready := make(chan chan struct{})
			ob := observable.New()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				es := test.makeStream(w, r)
				run := es.Start(ob, test.events...)

				if test.trigger != nil {
					c := <-ready
					close(c)
				}

				run()
			}))
			defer ts.Close()

			if test.trigger != nil {
				go func() {
					c := make(chan struct{})
					ready <- c
					<-c
					test.trigger(ob)
				}()
			}

			req, err := http.NewRequest("GET", ts.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(req.Context())
			defer cancel()

			req = req.WithContext(ctx)

			res, err := ts.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()

			test.respFunc(t, res)
		})
	}
}
