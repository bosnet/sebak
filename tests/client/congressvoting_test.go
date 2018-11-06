// +build client_integration_tests

package client

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/client"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

func TestCongressVoting(t *testing.T) {

	var (
		genesisAddr   = "GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ"
		genesisSecret = "SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN"
	)

	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	{
		genesisAccount, err := c.LoadAccount(genesisAddr)
		require.NoError(t, err)

		ob := operation.NewCongressVoting([]byte("dummy"), 10, 20)
		o, err := operation.NewOperation(ob)
		require.NoError(t, err)

		tx, err := transaction.NewTransaction(genesisAddr, uint64(genesisAccount.SequenceID), o)
		require.NoError(t, err)

		sender, err := keypair.Parse(genesisSecret)
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

		opage, err := c.LoadOperationsByAccount(genesisAddr, client.Q{Key: client.QueryType, Value: "congress-voting"})
		require.NoError(t, err)

		for _, obody := range opage.Embedded.Records {
			b, err := json.Marshal(obody.Body)
			require.NoError(t, err)
			var cv client.CongressVoting
			json.Unmarshal(b, &cv)
			require.Equal(t, ob.Contract, cv.Contract)
			require.Equal(t, ob.Voting.Start, cv.Voting.Start)
			require.Equal(t, ob.Voting.End, cv.Voting.End)
		}
	}
}

func TestCongressVotingResult(t *testing.T) {

	var (
		genesisAddr   = "GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ"
		genesisSecret = "SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN"
	)

	const (
		fee = 10000
	)

	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	{
		genesisAccount, err := c.LoadAccount(genesisAddr)
		require.NoError(t, err)

		ob := operation.NewCongressVotingResult(
			"dummy1",
			[]string{"http://localhost:12345/a", "http://localhost:12345/b"},
			"dummy2",
			[]string{"http://localhost:12345/c", "http://localhost:12345/d"},
			"dummy3",
			[]string{"http://localhost:12345/e", "http://localhost:12345/f"},
			100,
			70,
			20,
			10,
		)
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

		opage, err := c.LoadOperationsByAccount(genesisAddr, client.Q{Key: client.QueryType, Value: "congress-voting-result"})
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
			require.Equal(t, ob.TotalMembership, cvr.TotalMembership)
		}
	}
}
