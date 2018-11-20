// +build client_integration_tests

package client

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"boscoin.io/sebak/lib/client"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
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

	_, err = c.SubmitTransaction(body)
	require.NoError(t, err)

	var e error
	for second := time.Duration(0); second < time.Second*10; second = second + time.Millisecond*500 {
		_, e = c.LoadTransaction(tx.H.Hash)
		if e == nil {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
	require.Nil(t, e)

	var toAccount client.Account
	for second := time.Duration(0); second < time.Second*3; second = second + time.Millisecond*500 {
		toAccount, e = c.LoadAccount(toAddr)
		if e == nil {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
	require.Nil(t, e)

	targetBalance, err := strconv.ParseUint(toAccount.Balance, 10, 64)
	require.NoError(t, err)
	require.Equal(t, uint64(balance), targetBalance)

	fromAccount, err = c.LoadAccount(fromAddr)
	require.NoError(t, err)

	genesisBalance2, err := strconv.ParseUint(fromAccount.Balance, 10, 64)
	require.NoError(t, err)
	require.Equal(t, fromBalance-balance-fee, genesisBalance2)
}
