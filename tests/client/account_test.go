// +build client_integration_tests

package client

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/client"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node/runner/api/resource"
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

		ob := operation.NewCreateAccount(account1Addr, common.Amount(genesisToAccount1))
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(genesisAddr, uint64(genesisAccount.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(genesisSecret)
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

		var targetAccount client.Account
		for second := time.Duration(0); second < time.Second*3; second = second + time.Millisecond*500 {
			targetAccount, e = c.LoadAccount(account1Addr)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

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

		ob := operation.NewCreateAccount(account2Addr, common.Amount(account1ToAccount2))
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(account1Addr, uint64(senderAccount.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(account1Secret)
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

		var account2Account client.Account
		for second := time.Duration(0); second < time.Second*3; second = second + time.Millisecond*500 {
			account2Account, e = c.LoadAccount(account2Addr)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

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

		var account2Account client.Account
		for second := time.Duration(0); second < time.Second*3; second = second + time.Millisecond*500 {
			account2Account, e = c.LoadAccount(account2Addr)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

		targetBalance, err := strconv.ParseUint(account2Account.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, uint64(account1ToAccount2*2), targetBalance)

		senderAccount, err = c.LoadAccount(account1Addr)
		require.NoError(t, err)

		senderBalance2, err := strconv.ParseUint(senderAccount.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, senderBalance-account1ToAccount2-fee, senderBalance2)

	}

	//Payment from Account 1 to Account 2 with TransactionHistory
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

		pt, err := c.SubmitTransaction(body)
		require.Nil(t, err)

		for second := time.Duration(0); second < time.Second*10; second = second + time.Millisecond*500 {
			th, err := c.LoadTransactionHistory(pt.Hash)
			if err != nil {
				t.Log(err)
			}
			if th.Status == block.TransactionHistoryStatusConfirmed {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}

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

func TestFrozenAccount(t *testing.T) {

	var (
		genesisAddr   = "GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ"
		genesisSecret = "SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN"

		generalAccountAddr   = "GC4EWF2E2DPCQ5OL6EEWVCFRF5INTRB64WH4LYZW4KS7WWL6ZL5UCZ3D"
		generalAccountSecret = "SB4ZHXZXMRAGE54DAJJUAVQCI3B4ED5ZYQLDL43ZFWVGLSEHJIRKK2I3"
		frozenAccountAddr    = "GALDYMGQB2WOIH52LZ6YWH5LWBPT7AITQAJC6UXJCIOQ6N2DD4PRK4P5"
		frozenAccountSecret  = "SASABG3RYAIVXQKFL5LDR4EGPJT22NEQK6CPPYTENAYK6S7SYI2J3SUH"
	)

	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	common.UnfreezingPeriod = uint64(40)

	//Create from genesis to Account 1
	{
		const (
			genesisToGeneralAccount = 200000000000
		)

		genesisAccount, err := c.LoadAccount(genesisAddr)
		require.Nil(t, err)
		genesisBalance1, err := strconv.ParseUint(genesisAccount.Balance, 10, 64)
		require.Nil(t, err)

		ob := operation.NewCreateAccount(generalAccountAddr, common.Amount(genesisToGeneralAccount))
		o, err := operation.NewOperation(ob)
		require.Nil(t, err)

		tx, err := transaction.NewTransaction(genesisAddr, uint64(genesisAccount.SequenceID), o)
		require.Nil(t, err)

		sender, err := keypair.Parse(genesisSecret)
		require.Nil(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.Nil(t, err)

		_, err = c.SubmitTransaction(body)
		require.Nil(t, err)

		var e error
		for second := time.Duration(0); second < time.Second*10; second = second + time.Millisecond*500 {
			_, e = c.LoadTransaction(tx.H.Hash)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

		var targetAccount client.Account
		for second := time.Duration(0); second < time.Second*3; second = second + time.Millisecond*500 {
			targetAccount, e = c.LoadAccount(generalAccountAddr)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

		targetBalance, err := strconv.ParseUint(targetAccount.Balance, 10, 64)
		require.Nil(t, err)
		require.Equal(t, uint64(genesisToGeneralAccount), targetBalance)

		genesisAccount, err = c.LoadAccount(genesisAddr)
		require.Nil(t, err)

		genesisBalance2, err := strconv.ParseUint(genesisAccount.Balance, 10, 64)
		require.Nil(t, err)
		require.Equal(t, genesisBalance1-genesisToGeneralAccount-fee, genesisBalance2)
	}

	//Create from General Account to Frozen Account
	{
		const (
			generalAccountTofrozenAccount = 100000000000
		)

		senderAccount, err := c.LoadAccount(generalAccountAddr)
		require.Nil(t, err)
		senderBalance, err := strconv.ParseUint(senderAccount.Balance, 10, 64)

		ob := operation.NewFreezing(frozenAccountAddr, common.Amount(generalAccountTofrozenAccount), generalAccountAddr)
		o, err := operation.NewOperation(ob)
		require.Nil(t, err)

		tx, err := transaction.NewTransaction(generalAccountAddr, uint64(senderAccount.SequenceID), o)
		require.Nil(t, err)

		sender, err := keypair.Parse(generalAccountSecret)
		require.Nil(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.Nil(t, err)

		_, err = c.SubmitTransaction(body)
		require.Nil(t, err)

		var e error
		for second := time.Duration(0); second < time.Second*10; second = second + time.Millisecond*500 {
			_, e = c.LoadTransaction(tx.H.Hash)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

		var FrozenAccount1Account client.Account
		for second := time.Duration(0); second < time.Second*3; second = second + time.Millisecond*500 {
			FrozenAccount1Account, e = c.LoadAccount(frozenAccountAddr)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

		targetBalance, err := strconv.ParseUint(FrozenAccount1Account.Balance, 10, 64)
		require.Nil(t, err)
		require.Equal(t, uint64(generalAccountTofrozenAccount), targetBalance)

		senderAccount, err = c.LoadAccount(generalAccountAddr)
		require.Nil(t, err)

		senderBalance2, err := strconv.ParseUint(senderAccount.Balance, 10, 64)
		require.Nil(t, err)
		require.Equal(t, senderBalance-generalAccountTofrozenAccount, senderBalance2)

	}

	//UnfreezingRequest to Frozen Account
	{
		const (
			generalAccountTofrozenAccount = 100000000000
		)
		unfreezingAccount, err := c.LoadAccount(frozenAccountAddr)
		require.Nil(t, err)

		ob := operation.NewUnfreezeRequest()
		o, err := operation.NewOperation(ob)
		require.Nil(t, err)

		tx, err := transaction.NewTransaction(frozenAccountAddr, uint64(unfreezingAccount.SequenceID), o)
		require.Nil(t, err)

		sender, err := keypair.Parse(frozenAccountSecret)
		require.Nil(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.Nil(t, err)

		_, err = c.SubmitTransaction(body)
		require.Nil(t, err)

		var e error
		for second := time.Duration(0); second < time.Second*10; second = second + time.Millisecond*500 {
			_, e = c.LoadTransaction(tx.H.Hash)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

		var FrozenAccount2Account client.Account
		for second := time.Duration(0); second < time.Second*3; second = second + time.Millisecond*500 {
			FrozenAccount2Account, e = c.LoadAccount(frozenAccountAddr)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

		targetBalance, err := strconv.ParseUint(FrozenAccount2Account.Balance, 10, 64)
		require.Nil(t, err)
		require.Equal(t, uint64(generalAccountTofrozenAccount), targetBalance)

		var FrozenAccountsPage client.FrozenAccountsPage
		for second := time.Duration(0); second < time.Second*3; second = second + time.Millisecond*500 {
			FrozenAccountsPage, e = c.LoadFrozenAccountsByLinked(generalAccountAddr)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}

		require.Nil(t, e)
		require.Equal(t, FrozenAccountsPage.Embedded.Records[0].Address, frozenAccountAddr)
		require.Equal(t, FrozenAccountsPage.Embedded.Records[0].Amount, common.Amount(generalAccountTofrozenAccount))
		require.Equal(t, FrozenAccountsPage.Embedded.Records[0].Linked, generalAccountAddr)
		require.Equal(t, FrozenAccountsPage.Embedded.Records[0].State, resource.MeltingState)

	}

}
