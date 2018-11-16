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
	"boscoin.io/sebak/lib/block"
)

func createAccount(t *testing.T, fromAddr, fromSecret, toAddr string, balance uint64){

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

func TestInflationPF(t *testing.T) {

	const (
		genesisAddr   = "GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ"
		genesisSecret = "SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN"

		commonAddr   = "GCTVCU764UPXKRJW5DS5AWF5ETCCIYPZTWXN5U5CLUNAAJH6D5NGEIIH"
		commonSecret = "SDKQUVQWNQ3O7YBUBXHGZY6PE3WXIFEIW3Q67HLQRU2I4C7KEX5S62L2"

		CongressAddr   = "GAIJAH4FCEB3AWS2NVKRNDGVJ5QPD7VNOQNGZEPL2FAI6GPZQNKEXRWI"
		CongressSecret = "SBUONPFGBX7BF76ZZ2NOWWCJPG6NQQOQS37IL27JCUBPFAXFRYS3VDI2"

		account1Addr   = "GC6GJG5L6YPZQ6KP3HHD23UEAXZD22YLD4LUV7PZPGLNFW3MR6K7H6PX"
		account1Secret = "SCEAY3J3W7B4LN5O3LTFFCYHBX73E7DAGPSWUW4V74UYQTKZAXTUJOMR"

		payAmount = 100000000

		fundingAmount = 123456789
	)

	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	// Prepare Congress Address, and Funder Address
	{
		_, err := c.LoadAccount(CongressAddr)
		if err != nil {
			//Create from genesis to Congress if not exists
			createAccount(t, genesisAddr, genesisSecret, CongressAddr, payAmount)
		}

		createAccount(t, genesisAddr, genesisSecret, account1Addr, payAmount)
	}

	// Congress Voting
	{
		congressAccount, err := c.LoadAccount(CongressAddr)
		require.NoError(t, err)

		ob := operation.NewCongressVoting([]byte("dummy"), 10, 20, common.Amount(fundingAmount), account1Addr)
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(CongressAddr, uint64(congressAccount.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(CongressSecret)
		require.NoError(t, err)
		tx.Sign(sender, []byte(NETWORK_ID))

		body, err := tx.Serialize()
		require.NoError(t, err)

		pt, err := c.SubmitTransaction(body)
		require.NoError(t, err)
		require.Equal(t, pt.Status, block.TransactionHistoryStatusSubmitted)

		var e error
		for second := time.Duration(0); second < time.Second*10; second = second + time.Millisecond*500 {
			_, e = c.LoadTransaction(tx.H.Hash)
			if e == nil {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
		require.Nil(t, e)

		opage, err := c.LoadOperationsByAccount(CongressAddr, client.Q{Key: client.QueryType, Value: "congress-voting"})
		require.NoError(t, err)

		for _, obody := range opage.Embedded.Records {
			b, err := json.Marshal(obody.Body)
			require.NoError(t, err)
			var cv client.CongressVoting
			json.Unmarshal(b, &cv)
			require.Equal(t, ob.Contract, cv.Contract)
			require.Equal(t, ob.Voting.Start, cv.Voting.Start)
			require.Equal(t, ob.Voting.End, cv.Voting.End)
			require.Equal(t, ob.FundingAddress, cv.FundingAddress)
			require.Equal(t, ob.Amount, cv.Amount)
		}
	}

	// Congress Voting Result
	{
		congressAccount, err := c.LoadAccount(CongressAddr)
		require.NoError(t, err)

		oPage, err := c.LoadOperationsByAccount(CongressAddr, client.Q{Key: client.QueryType, Value: string(operation.TypeCongressVoting)})
		require.NoError(t, err)

		oHash := oPage.Embedded.Records[0].Hash

		ob := operation.NewCongressVotingResult(
			"dummy1",
			[]string{"a", "b"},
			"dummy2",
			[]string{"c", "d"},
			100,
			70,
			20,
			10,
			oHash,
		)
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(CongressAddr, congressAccount.SequenceID, o)
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
			require.Equal(t, ob.CongressVotingHash, cvr.CongressVotingHash)
		}
	}

	// PF Inflation Operation.
	{

		commonAccount, err := c.LoadAccount(commonAddr)
		require.NoError(t, err)

		var targetAccount client.Account
		targetAccount, err = c.LoadAccount(account1Addr)
		require.Nil(t, err)

		beforeTargetAmount, err := strconv.ParseUint(targetAccount.Balance, 10, 64)
		require.NoError(t, err)

		oPage, err := c.LoadOperationsByAccount(CongressAddr, client.Q{Key: client.QueryType, Value: string(operation.TypeCongressVotingResult)})
		require.NoError(t, err)

		oHash := oPage.Embedded.Records[0].Hash

		ob := operation.NewInflationPF(account1Addr, common.Amount(fundingAmount), oHash)
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

		targetAccount, err = c.LoadAccount(account1Addr)
		require.Nil(t, err)

		targetBalance, err := strconv.ParseUint(targetAccount.Balance, 10, 64)
		require.NoError(t, err)
		require.Equal(t, uint64(fundingAmount), targetBalance -beforeTargetAmount)

	}
}
