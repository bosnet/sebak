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

func TestTransactionStatus(t *testing.T) {

	const (
		genesisAddr   = "GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ"
		genesisSecret = "SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN"

		account1Addr   = "GAC6Q6TRNOQJLSFNGVSSAGDFLLYNONWJSUMLNX3FZFMWSA5TABELHN54"
		account1Secret = "SDJA7RJZUE4MJ3NAAHCYDXU54XI5W4ERM4CFY5PICBUDOB7Z6HW44STA"
	)

	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	//Create from genesis to Account 1
	{
		const (
			genesisToAccount1 = 100000000
		)

		genesisAccount, err := c.LoadAccount(genesisAddr)
		require.NoError(t, err)
		genesisBalance, err := strconv.ParseUint(genesisAccount.Balance, 10, 64)
		require.NoError(t, err)

		ob := operation.NewCreateAccount(account1Addr, common.Amount(genesisToAccount1), "")
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(genesisAddr, uint64(genesisAccount.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(genesisSecret)
		require.NoError(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.NoError(t, err)

		_, err = c.SubmitTransactionAndWait(tx.H.Hash, body)
		require.NoError(t, err)

		targetAccount, err := c.LoadAccount(account1Addr)
		require.NoError(t, err)

		targetBalance, err := strconv.ParseUint(targetAccount.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, uint64(genesisToAccount1), targetBalance)

		genesisAccount, err = c.LoadAccount(genesisAddr)
		require.NoError(t, err)

		genesisBalance2, err := strconv.ParseUint(genesisAccount.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, genesisBalance-genesisToAccount1-fee, genesisBalance2)
	}
}
