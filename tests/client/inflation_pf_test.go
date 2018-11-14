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
	"encoding/json"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestInflationPF(t *testing.T) {

	const (
		genesisAddr   = "GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ"
		genesisSecret = "SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN"

		commonAddr   = "GCTVCU764UPXKRJW5DS5AWF5ETCCIYPZTWXN5U5CLUNAAJH6D5NGEIIH"
		commonSecret = "SDKQUVQWNQ3O7YBUBXHGZY6PE3WXIFEIW3Q67HLQRU2I4C7KEX5S62L2"

		CongressAddr   = "GAIJAH4FCEB3AWS2NVKRNDGVJ5QPD7VNOQNGZEPL2FAI6GPZQNKEXRWI"
		CongressSecret = "SBUONPFGBX7BF76ZZ2NOWWCJPG6NQQOQS37IL27JCUBPFAXFRYS3VDI2"
	)

	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	// Prepare Voting Result Operation
	{
		_, err := c.LoadAccount(CongressAddr)
		if err != nil {
			//Create from genesis to Congress if not exists
			{
				const (
					genesisToCongressAddr = 100000000
				)

				genesisAccount, err := c.LoadAccount(genesisAddr)
				require.NoError(t, err)

				ob := operation.NewCreateAccount(CongressAddr, common.Amount(genesisToCongressAddr), "")
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

				for second := time.Duration(0); second < time.Second*3; second = second + time.Millisecond*500 {
					_, e = c.LoadAccount(CongressAddr)
					if e == nil {
						break
					}
					time.Sleep(time.Millisecond * 500)
				}
				require.Nil(t, e)
			}
		}

		ob := operation.NewCongressVotingResult(
			"dummy1",
			[]string{"a", "b"},
			"dummy2",
			[]string{"c", "d"},
			100,
			70,
			20,
			10,
		)
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(CongressAddr, uint64(0), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(CongressSecret)
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

		opage, err := c.LoadOperationsByAccount(CongressAddr, client.Q{Key: client.QueryType, Value: "congress-voting-result"})
		require.NoError(t, err)

		for _, obody := range opage.Embedded.Records {
			b, err := json.Marshal(obody.Body)
			require.NoError(t, err)
			var cvr client.CongressVotingResult
			json.Unmarshal(b, &cvr)
			require.Equal(t, ob.BallotStamps.Hash, cvr.BallotStamps.Hash)
			require.Equal(t, ob.BallotStamps.Urls, cvr.BallotStamps.Urls)
			require.Equal(t, ob.Voters.Hash, cvr.Voters.Hash)
			require.Equal(t, ob.Voters.Urls, cvr.Voters.Urls)
			require.Equal(t, ob.Result.Count, cvr.Result.Count)
			require.Equal(t, ob.Result.Yes, cvr.Result.Yes)
			require.Equal(t, ob.Result.No, cvr.Result.No)
			require.Equal(t, ob.Result.ABS, cvr.Result.ABS)
		}
	}

	// PF Inflation Operation.
	{
		const (
			pfInflationAmount = 123456789
		)

		commonAccount, err := c.LoadAccount(commonAddr)
		require.NoError(t, err)
		beforeCommonAmount, err := strconv.ParseUint(commonAccount.Balance, 10, 64)
		require.NoError(t, err)
		beforeCommonAmount %= 1000000000 // remove coinbase

		oPage, err := c.LoadOperationsByAccount(CongressAddr, client.Q{Key: client.QueryType, Value: string(operation.TypeCongressVotingResult)})
		require.NoError(t, err)

		oHash := oPage.Embedded.Records[0].Hash

		ob := operation.NewInflationPF(common.Amount(pfInflationAmount), oHash)
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(commonAddr, uint64(commonAccount.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(commonSecret)
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
			targetAccount, e = c.LoadAccount(commonAddr)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

		targetBalance, err := strconv.ParseUint(targetAccount.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, uint64(pfInflationAmount), targetBalance%1000000000-beforeCommonAmount)

	}
}
