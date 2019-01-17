// +build client_integration_tests

package client

import (
	"boscoin.io/sebak/lib/client"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
	"net/http"
	"strconv"
	"testing"
)

func createAccount(t *testing.T, fromAddr, fromSecret, toAddr string, balance uint64) {

	c := client.MustNewClient("https://127.0.0.1:2830")
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

	_, err = c.SubmitTransactionAndWait(tx.H.Hash, body)
	require.NoError(t, err)

	toAccount, err := c.LoadAccount(toAddr)
	require.NoError(t, err)
	targetBalance, err := strconv.ParseUint(toAccount.Balance, 10, 64)
	require.NoError(t, err)
	require.Equal(t, uint64(balance), targetBalance)

	fromAccount, err = c.LoadAccount(fromAddr)
	fromBalance2, err := strconv.ParseUint(fromAccount.Balance, 10, 64)
	require.NoError(t, err)
	require.Equal(t, fromBalance-balance-fee, fromBalance2)
}
