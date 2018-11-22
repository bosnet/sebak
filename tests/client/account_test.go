// +build client_integration_tests

package client

import (
	"net/http"
	"strconv"
	"testing"

	"boscoin.io/sebak/lib/client"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestAccount(t *testing.T) {

	const (
		genesisAddr   = "GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ"
		genesisSecret = "SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN"

		account1Addr   = "GAVDK2OHFZ5B257PRTCOFYNGRIWV5JRCD5SINMLQJUMSSVYV4LVHI4CN"
		account1Secret = "SDNKCPIVRCS76DATVQUFXDO73DPSXVJ22YCIS46JOBV3UR47ONWFKEUX"
		account2Addr   = "GANCZWXAJWFBZJ3NDOSCJSSNOEARMRXMOV4RXWJ6PLPLPJWT6CELZJCS"
		//account2Secret = "SBOEFVTSQCFFTHHFAIPLOBMDY32JC4E4KEHR4TKCSUE2O5BSBTHOAANH"
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

	//Create from Account 1 to Account 2
	{
		const (
			account1ToAccount2 = 1000000
		)

		senderAccount, err := c.LoadAccount(account1Addr)
		require.NoError(t, err)
		senderBalance, err := strconv.ParseUint(senderAccount.Balance, 10, 64)

		ob := operation.NewCreateAccount(account2Addr, common.Amount(account1ToAccount2), "")
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(account1Addr, uint64(senderAccount.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(account1Secret)
		require.NoError(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.NoError(t, err)

		_, err = c.SubmitTransactionAndWait(tx.H.Hash, body)
		require.NoError(t, err)

		account2Account, err := c.LoadAccount(account2Addr)
		require.NoError(t, err)

		targetBalance, err := strconv.ParseUint(account2Account.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, uint64(account1ToAccount2), targetBalance)

		senderAccount, err = c.LoadAccount(account1Addr)
		require.NoError(t, err)

		senderBalance2, err := strconv.ParseUint(senderAccount.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, senderBalance-account1ToAccount2-fee, senderBalance2)

	}

	//Payment from Account 1 to Account 2
	{
		const (
			account1ToAccount2 = 1000000
		)

		senderAccount, err := c.LoadAccount(account1Addr)
		require.NoError(t, err)
		senderBalance, err := strconv.ParseUint(senderAccount.Balance, 10, 64)

		ob := operation.NewPayment(account2Addr, common.Amount(account1ToAccount2))
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(account1Addr, uint64(senderAccount.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(account1Secret)
		require.NoError(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.NoError(t, err)

		_, err = c.SubmitTransactionAndWait(tx.H.Hash, body)
		require.NoError(t, err)

		account2Account, err := c.LoadAccount(account2Addr)
		require.NoError(t, err)

		targetBalance, err := strconv.ParseUint(account2Account.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, uint64(account1ToAccount2*2), targetBalance)

		senderAccount, err = c.LoadAccount(account1Addr)
		require.NoError(t, err)

		senderBalance2, err := strconv.ParseUint(senderAccount.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, senderBalance-account1ToAccount2-fee, senderBalance2)

	}

	//Payment from Account 1 to Account 2 with TransactionStatus
	{
		const (
			account1ToAccount2 = 1000000
		)

		senderAccount, err := c.LoadAccount(account1Addr)
		require.Nil(t, err)
		senderBalance, err := strconv.ParseUint(senderAccount.Balance, 10, 64)

		ob := operation.NewPayment(account2Addr, common.Amount(account1ToAccount2))
		o, err := operation.NewOperation(ob)
		require.Nil(t, err)

		tx, err := transaction.NewTransaction(account1Addr, uint64(senderAccount.SequenceID), o)
		require.Nil(t, err)

		sender, err := keypair.Parse(account1Secret)
		require.Nil(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.Nil(t, err)

		_, err = c.SubmitTransactionAndWait(tx.H.Hash, body)
		require.Nil(t, err)

		account2Account, err := c.LoadAccount(account2Addr)
		require.Nil(t, err)

		targetBalance, err := strconv.ParseUint(account2Account.Balance, 10, 64)
		require.Nil(t, err)
		require.Equal(t, uint64(account1ToAccount2*3), targetBalance)

		senderAccount, err = c.LoadAccount(account1Addr)
		require.Nil(t, err)

		senderBalance2, err := strconv.ParseUint(senderAccount.Balance, 10, 64)
		require.Nil(t, err)
		require.Equal(t, senderBalance-account1ToAccount2-fee, senderBalance2)

	}

}
