// +build client_integration_tests

package client

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"boscoin.io/sebak/lib/client"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestFreezingAccount(t *testing.T) {

	const (
		genesisAddr   = "GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ"
		genesisSecret = "SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN"

		account1Addr   = "GA7LUSDRZGIXFQXB6QA4NAWWRI6K32TFQD23WRRBHICUH2SPPT372TM4"
		account1Secret = "SAO6ENZWXIFFNFTSNLPN2VJCEOH5AZ7F6YJZYVEX2VRW5U26VQ4C7M6L"

		account2Addr   = "GDELVSEXHKACSDMKTWIUM5D2XC55XRKQQGIIWYXNOCMMDX226UA6YRRO"
		account2Secret = "SC6X7ZH3OD77ZXLYZ32MJLGVIOPUCMMACI35HJMHX2PPCZXGLRVTMUPA"
	)

	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	// Prepare Congress Address, and Funder Address
	{
		createAccount(t, genesisAddr, genesisSecret, account1Addr, uint64(common.Unit))
	}

	// Freezing
	{
		account1Account, err := c.LoadAccount(account1Addr)
		require.NoError(t, err)

		ob := operation.NewCreateAccount(account2Addr, common.Unit, account1Addr)
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(account1Addr, uint64(account1Account.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(account1Secret)
		require.NoError(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)

		var account2Account client.Account
		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			err = c.StreamAccount(ctx, account2Addr, func(account client.Account) {
				if account.Address != "" {
					account2Account = account
					cancel()
				}
			})
			require.NoError(t, err)
			wg.Done()
		}()

		_, err = c.SubmitTransactionAndWait(tx.H.Hash, body)
		require.NoError(t, err)

		wg.Wait()

		require.NoError(t, err)
		account2Amount, err := common.AmountFromString(account2Account.Balance)
		require.NoError(t, err)

		require.Equal(t, common.Unit, account2Amount)
	}

	// UnFreezing
	{
		account2Account, err := c.LoadAccount(account2Addr)
		require.NoError(t, err)

		ob := operation.NewUnfreezeRequest()
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(account2Addr, uint64(account2Account.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(account2Secret)
		require.NoError(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.NoError(t, err)

		_, err = c.SubmitTransactionAndWait(tx.H.Hash, body)
		require.NoError(t, err)

		account2Account, err = c.LoadAccount(account2Addr)
		require.NoError(t, err)
		account2Amount, err := common.AmountFromString(account2Account.Balance)
		require.NoError(t, err)

		require.Equal(t, common.Unit, account2Amount)
	}

	// Refund
	{

		time.Sleep(time.Second * 10)
		account2Account, err := c.LoadAccount(account2Addr)
		require.NoError(t, err)

		ob := operation.NewPayment(account1Addr, common.Unit.MustSub(common.Amount(fee)))
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(account2Addr, uint64(account2Account.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(account2Secret)
		require.NoError(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.NoError(t, err)

		account1Account, err := c.LoadAccount(account1Addr)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			err = c.StreamAccount(ctx, account1Addr, func(account client.Account) {
				if account1Account.Balance != account.Balance {
					account1Account = account
					cancel()
				}
			})
			require.NoError(t, err)
			wg.Done()
		}()

		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			err = c.StreamAccount(ctx, account2Addr, func(account client.Account) {
				if account2Account.Balance != account.Balance {
					account2Account = account
					cancel()
				}
			})
			require.NoError(t, err)
			wg.Done()
		}()

		_, err = c.SubmitTransactionAndWait(tx.H.Hash, body)
		require.NoError(t, err)

		wg.Wait()

		account1Amount, err := common.AmountFromString(account1Account.Balance)
		require.NoError(t, err)

		require.Equal(t, common.Unit.MustSub(common.Amount(fee)), account1Amount)

		account2Amount, err := common.AmountFromString(account2Account.Balance)
		require.NoError(t, err)

		require.Equal(t, common.Amount(0), account2Amount)

	}

}
