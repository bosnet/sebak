package sebak

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"boscoin.io/sebak/lib/block"
	observable "github.com/GianlucaGuarini/go-observable"

	"github.com/stretchr/testify/require"
)

func TestAPIStreamRun(t *testing.T) {
	tests := []struct {
		name       string
		makeStream func(*observable.Observable) *EventStream
		trigger    func(*observable.Observable)
		respFunc   func(testing.TB, *http.Response)
	}{
		{
			"default",
			func(ob *observable.Observable) *EventStream {
				es := NewEventStream(ob, "test1")
				return es
			},
			func(ob *observable.Observable) {
				ob.Trigger("test1", block.NewBlockAccount("hello", 100, "tx1-tx1"))
			},
			func(t testing.TB, res *http.Response) {
				s := bufio.NewScanner(res.Body)
				s.Scan()

				var ba block.BlockAccount
				require.Nil(t, json.Unmarshal(s.Bytes(), &ba))
				require.Nil(t, s.Err())
				require.Equal(t, ba, *block.NewBlockAccount("hello", 100, "tx1-tx1"))
			},
		},
		{
			"onFunc",
			func(ob *observable.Observable) *EventStream {
				es := NewEventStream(ob, "test1")
				es.On(func(args ...interface{}) ([]byte, error) {
					s, ok := args[1].(*block.BlockAccount)
					if !ok {
						return nil, fmt.Errorf("this is not serializable")
					}
					bs, err := s.Serialize()
					if err != nil {
						return nil, err
					}
					return bs, nil

				})
				return es
			},
			func(ob *observable.Observable) {
				ob.Trigger("test1", block.NewBlockAccount("hello", 100, "tx1-tx1"))
			},
			func(t testing.TB, res *http.Response) {
				s := bufio.NewScanner(res.Body)
				s.Scan()

				var ba block.BlockAccount
				require.Nil(t, json.Unmarshal(s.Bytes(), &ba))
				require.Nil(t, s.Err())
				require.Equal(t, ba, *block.NewBlockAccount("hello", 100, "tx1-tx1"))
			},
		},
		{
			"beforeFunc",
			func(ob *observable.Observable) *EventStream {
				es := NewEventStream(ob, "test1")
				es.Before(func() {
					ob.Trigger("test1", block.NewBlockAccount("hello", 100, "tx1-tx1"))
				})
				return es
			},
			nil,
			func(t testing.TB, res *http.Response) {
				s := bufio.NewScanner(res.Body)
				s.Scan()

				var ba block.BlockAccount
				require.Nil(t, json.Unmarshal(s.Bytes(), &ba))
				require.Nil(t, s.Err())
				require.Equal(t, ba, *block.NewBlockAccount("hello", 100, "tx1-tx1"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ready := make(chan chan struct{})
			ob := observable.New()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				es := test.makeStream(ob)
				run := es.Start(w, r)

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
