// +build client_integration_tests

package client

import (
	"boscoin.io/sebak/lib/client"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"context"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
	"net/http"
	"strconv"
	"sync"
	"testing"
)

func createAccount(t *testing.T, fromAddr, fromSecret, toAddr string, balance uint64) {

	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	fromAccount, err := c.LoadAccount(fromAddr)
	require.NoError(t, err)
	fromBalance, err := strconv.ParseUint(fromAccount.Balance, 10, 64)
	require.NoError(t, err)

	ob := operation.NewCreateAccount(toAddr, common.Amount(balance), "")
	o, err := operation.NewOperation(ob)
	require.NoError(t, err)

	tx, err := transaction.NewTransaction(fromAddr, uint64(fromAccount.SequenceID), o)
	require.NoError(t, err)

	sender, err := keypair.Parse(fromSecret)
	require.NoError(t, err)
	tx.Sign(sender, []byte(NETWORK_ID))

	body, err := tx.Serialize()
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		err = c.StreamTransactionStatus(ctx, tx.H.Hash, nil, func(status client.TransactionStatus) {
			if status.Status == "confirmed" {
				cancel()
			}
		})
		require.NoError(t, err)
		wg.Done()
	}()

	var toAccount client.Account
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		err = c.StreamAccount(ctx, toAddr, nil, func(account client.Account) {
			toAccount = account
			cancel()
		})
		require.NoError(t, err)
		wg.Done()
	}()

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		err = c.StreamAccount(ctx, fromAddr, nil, func(account client.Account) {
			if account.SequenceID != fromAccount.SequenceID {
				fromAccount = account
				cancel()
			}
		})
		require.NoError(t, err)
		wg.Done()
	}()

	_, err = c.SubmitTransaction(body)
	require.NoError(t, err)

	wg.Wait()

	targetBalance, err := strconv.ParseUint(toAccount.Balance, 10, 64)
	require.NoError(t, err)
	require.Equal(t, uint64(balance), targetBalance)

	genesisBalance2, err := strconv.ParseUint(fromAccount.Balance, 10, 64)
	require.NoError(t, err)
	require.Equal(t, fromBalance-balance-fee, genesisBalance2)
}
