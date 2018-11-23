// +build client_integration_tests

package client

import (
	"boscoin.io/sebak/lib/client"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestOperation(t *testing.T) {

	const (
		genesisAddr   = "GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ"
		genesisSecret = "SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN"

		account1Addr   = "GASB5RUQZB2VHM7Z25UEM374I655CUJK2OXFTNW275BZO5GEJULLSOIS"
		account1Secret = "SBXVOPDRVGUWDRUDKWSH4PGBAHMC7FTPFQQKZ5MLBYQWROVNY73N3YIX"
		account2Addr   = "GAATM6UE2OJEISHYOPWPGU3BXZY766MEZR7VKLD65DF4UHDEDLIGDCXB"
		//account2Secret = "SAUR6KPJY6GT7FQYDLYNXHHLIVPNA4JSJ4IABNBVYUBLZD7LCZM5KQPY"
	)

	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	//Create from genesis to Account 1
	{
		const (
			createBalance      = 10000000000
			account1To2Balance = createBalance / 10
		)
		createAccount(t, genesisAddr, genesisSecret, account1Addr, createBalance)
		createAccount(t, account1Addr, account1Secret, account2Addr, account1To2Balance)
	}

	{
		opage, err := c.LoadOperationsByAccount(account1Addr)
		require.NoError(t, err)
		for _, op := range opage.Embedded.Records {
			if op.Source == account1Addr {
				require.Equal(t, op.Target, account2Addr)
			}
			if op.Target == account1Addr {
				require.Equal(t, op.Source, genesisAddr)
			}
		}
	}

	{
		opage, err := c.LoadOperationsByAccount(account2Addr)
		require.NoError(t, err)
		for _, op := range opage.Embedded.Records {
			if op.Target == account2Addr {
				require.Equal(t, op.Source, account1Addr)
			}
		}
	}
}
