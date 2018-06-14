package sebak

/*
import "sync"

import (
	"boscoin.io/sebak/lib/storage"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestStreaming(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(1)

	// Setting Server
	storageConfig, err := sebakstorage.NewConfigFromString("memory://")
	if err != nil {
		t.Errorf("failed to initialize file db: %v", err)

	}
	storage, err := sebakstorage.NewStorage(storageConfig)
	if err != nil {
		t.Errorf("failed to initialize file db: %v", err)
	}
	defer storage.Close()

	ctx := context.WithValue(context.Background(), "storage", storage)
	router := mux.NewRouter()
	router.HandleFunc("/account/{address}", GetAccountHandler(ctx)).Methods("GET")
	server := &http.Server{Addr: ":5000", Handler: router}
	go server.ListenAndServe()

	// Make Dummy BlockAccount
	ba := testMakeBlockAccount()
	ba.Save(storage)
	prev := ba.GetBalance()

	// Do stream Request to the Server
	go func() {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:5000/account/%s", ba.Address), nil)
		if err != nil {

		}
		req.Header.Set("Accept", "text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error("Server not yet initialized")
		}

		if err != nil {
			for {
				resp, err = http.DefaultClient.Do(req)
				if err == nil {
					break
				}
				time.Sleep(time.Second)
			}
		}
		reader := bufio.NewReader(resp.Body)
		var n Amount
		for n = 1; n < 10 ; n++ {
			line, _ := reader.ReadBytes('\n')
			var cba = &BlockAccount{}
			json.Unmarshal(line, cba)

			if ba.Address != cba.Address {
				t.Errorf("Expected:%s Actual:%s", ba.Address, cba.Address)
			}
			if cba.GetBalance()-prev != n {
				t.Errorf("Expected:%d Actual:%d", prev+n, cba.GetBalance())
			}
			prev = cba.GetBalance()
		}
		resp.Body.Close()
		wg.Done()
	}()

	// Makes Some Events
	for n := 1; n < 20; n++ {
		newBalance, _ := ba.GetBalance().Add(Amount(n))
		ba.Balance = newBalance.String()

		ba.Save(storage)
		time.Sleep(time.Millisecond * 100)
	}

	wg.Wait()
	server.Shutdown(context.Background())
}
*/
