package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	cmdcommon "boscoin.io/sebak/cmd/sebak/common"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/sync"
)

func TestParseFlagValidators(t *testing.T) {
	vs, err := parseFlagValidators("https://localhost:12346?address=GDPQ2LBYP3RL3O675H2N5IEYM6PRJNUA5QFMKXIHGTKEB5KS5T3KHFA2")
	require.NoError(t, err)
	require.Equal(t, 1, len(vs))
}

func TestParseFlagsNode(t *testing.T) {
	flagNetworkID = "sebak-test-network"
	flagValidators = "https://localhost:12346?address=GDPQ2LBYP3RL3O675H2N5IEYM6PRJNUA5QFMKXIHGTKEB5KS5T3KHFA2"
	flagKPSecretSeed = "SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM"
	flagBindURL = "http://0.0.0.0:12345"

	flagBlockTime = "200ms"
	flagBlockTimeDelta = "50ms"
	flagTimeoutINIT = "100ms"
	flagTimeoutSIGN = "50ms"
	flagTimeoutACCEPT = "50ms"
	flagTimeoutALLCONFIRM = "10s"

	parseFlagsNode()

	require.Equal(t, 200*time.Millisecond, blockTime)
	require.Equal(t, 50*time.Millisecond, blockTimeDelta)
	require.Equal(t, 100*time.Millisecond, timeoutINIT)
	require.Equal(t, 50*time.Millisecond, timeoutSIGN)
	require.Equal(t, 50*time.Millisecond, timeoutACCEPT)
	require.Equal(t, 10*time.Second, timeoutALLCONFIRM)

	require.Equal(t, uint64(common.DefaultTransactionsInBallotLimit), transactionsLimit)
	require.Equal(t, uint64(common.DefaultOperationsInTransactionLimit), operationsLimit)
	require.Equal(t, uint64(common.DefaultOperationsInBallotLimit), operationsInBallotLimit)

	require.Equal(t, 67, threshold)

	require.Equal(t, "100-M", rateLimitRuleAPI.Default.Formatted)
	require.Equal(t, "100-S", rateLimitRuleNode.Default.Formatted)

	require.Equal(t, uint64(1000000), txPoolClientLimit)
	require.Equal(t, uint64(0) /* unlimited */, txPoolNodeLimit)

	require.Equal(t, uint64(241920), common.UnfreezingPeriod)
	require.Equal(t, uint64(300), syncPoolSize)

	require.Equal(t, sync.RetryInterval, syncRetryInterval)
	require.Equal(t, sync.FetchTimeout, syncFetchTimeout)
	require.Equal(t, sync.CheckBlockHeightInterval, syncCheckInterval)
	require.Equal(t, sync.CheckPrevBlockInterval, syncCheckPrevBlock)
	require.Equal(t, sync.WatchInterval, watchInterval)
}

func TestParseFlagSelfValidators(t *testing.T) {
	flagNetworkID = "sebak-test-network"
	flagValidators = "https://localhost:12346?address=GDPQ2LBYP3RL3O675H2N5IEYM6PRJNUA5QFMKXIHGTKEB5KS5T3KHFA2"
	flagKPSecretSeed = "SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM"
	flagBindURL = "http://0.0.0.0:12345"

	parseFlagsNode()
	require.Equal(t, 2, len(localNode.GetValidators()))

	parsedValidator, _ := node.NewValidatorFromURI(flagValidators)
	validator := localNode.GetValidators()[parsedValidator.Address()]

	require.Equal(t, validator.Address(), parsedValidator.Address())
	require.Equal(t, validator.Endpoint().Host, parsedValidator.Endpoint().Host)
	require.Equal(t, validator.Endpoint().Port(), parsedValidator.Endpoint().Port())
}

func TestAddingSelfValidatorsWithoutSelf(t *testing.T) {
	flagNetworkID = "sebak-test-network"
	flagValidators = "https://localhost:12346?address=GDPQ2LBYP3RL3O675H2N5IEYM6PRJNUA5QFMKXIHGTKEB5KS5T3KHFA2"
	flagKPSecretSeed = "SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM"
	flagBindURL = "http://0.0.0.0:12345"

	parseFlagsNode()

	require.NotNil(t, localNode)
	require.Equal(t, 2, len(localNode.GetValidators()))

	{ // check validator added
		var found bool
		v, _ := node.NewValidatorFromURI(flagValidators)
		for _, validator := range localNode.GetValidators() {
			if v.Address() == validator.Address() {
				found = true
			}
		}
		require.True(t, found)
	}

	{ // check LocalNode added
		var found bool
		for _, validator := range localNode.GetValidators() {
			if localNode.Address() == validator.Address() {
				found = true
			}
		}
		require.True(t, found)
	}
}

func TestAddingSelfValidatorsWithSelf(t *testing.T) {
	targetValidators := "https://localhost:12346?address=GDPQ2LBYP3RL3O675H2N5IEYM6PRJNUA5QFMKXIHGTKEB5KS5T3KHFA2"

	flagNetworkID = "sebak-test-network"
	flagValidators = "self " + targetValidators
	flagKPSecretSeed = "SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM"
	flagBindURL = "http://0.0.0.0:12345"

	parseFlagsNode()

	require.NotNil(t, localNode)
	require.Equal(t, 2, len(localNode.GetValidators()))

	{ // check validator added
		var found bool
		v, _ := node.NewValidatorFromURI(targetValidators)
		for _, validator := range localNode.GetValidators() {
			if v.Address() == validator.Address() {
				found = true
			}
		}
		require.True(t, found)
	}

	{ // check LocalNode added
		var found bool
		for _, validator := range localNode.GetValidators() {
			if localNode.Address() == validator.Address() {
				found = true
			}
		}
		require.True(t, found)
	}
}

func TestAddingSelfValidatorsWithOnlySelf(t *testing.T) {
	flagNetworkID = "sebak-test-network"
	flagValidators = "self"
	flagKPSecretSeed = "SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM"
	flagBindURL = "http://0.0.0.0:12345"

	parseFlagsNode()

	require.NotNil(t, localNode)
	require.Equal(t, 1, len(localNode.GetValidators()))

	validator, ok := localNode.GetValidators()[localNode.Address()]
	require.True(t, ok)
	require.Equal(t, localNode.Address(), validator.Address())
}

func TestAddingSelfWithoutValidators(t *testing.T) {
	flagNetworkID = "sebak-test-network"
	flagValidators = ""
	flagKPSecretSeed = "SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM"
	flagBindURL = "http://0.0.0.0:12345"
	_, err := parseFlagValidators(flagValidators)
	require.Error(t, err)
}

func TestParseFlagRateLimit(t *testing.T) {
	testFunc := func(cmdline string) (common.RateLimitRule, error) {
		testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

		var fr cmdcommon.ListFlags
		testCmd.Var(&fr, "rate-limit-api", "")

		err := testCmd.Parse(strings.Fields(cmdline))
		require.NoError(t, err)

		return parseFlagRateLimit(fr, common.RateLimitAPI)
	}

	{ // weird value
		_, err := testFunc("--rate-limit-api=showme")
		require.Error(t, err)
	}

	{ // valid value
		rule, err := testFunc("--rate-limit-api=10-S")
		require.NoError(t, err)
		require.Equal(t, time.Second, rule.Default.Period)
		require.Equal(t, int64(10), rule.Default.Limit)
		require.Equal(t, 0, len(rule.ByIPAddress))
	}

	{ // multiple value, last will be choose.
		rule, err := testFunc("--rate-limit-api=10-S --rate-limit-api=9-M")
		require.NoError(t, err)
		require.Equal(t, time.Minute, rule.Default.Period)
		require.Equal(t, int64(9), rule.Default.Limit)
		require.Equal(t, 0, len(rule.ByIPAddress))
	}

	{ // with ip address, but `common.RateLimitAPI` will be default
		const allowedIP = "1.2.3.4"
		rule, err := testFunc("--rate-limit-api=" + allowedIP + "=8-S")
		require.NoError(t, err)
		require.Equal(t, common.RateLimitAPI.Period, rule.Default.Period)
		require.Equal(t, common.RateLimitAPI.Limit, rule.Default.Limit)
		require.Equal(t, 1, len(rule.ByIPAddress))
		require.NotNil(t, rule.ByIPAddress[allowedIP])
		require.Equal(t, time.Second, rule.ByIPAddress[allowedIP].Period)
		require.Equal(t, int64(8), rule.ByIPAddress[allowedIP].Limit)
	}

	{ // with ip address and with default
		const allowedIP = "1.2.3.4"
		rule, err := testFunc("--rate-limit-api=11-H --rate-limit-api=" + allowedIP + "=8-S")
		require.NoError(t, err)
		require.Equal(t, time.Hour, rule.Default.Period)
		require.Equal(t, int64(11), rule.Default.Limit)
		require.Equal(t, 1, len(rule.ByIPAddress))
		require.NotNil(t, rule.ByIPAddress[allowedIP])
		require.Equal(t, time.Second, rule.ByIPAddress[allowedIP].Period)
		require.Equal(t, int64(8), rule.ByIPAddress[allowedIP].Limit)
	}

	{ // unlimit
		rule, err := testFunc("--rate-limit-api=0-S")
		require.NoError(t, err)
		require.Equal(t, time.Second, rule.Default.Period)
		require.Equal(t, int64(0), rule.Default.Limit)
		require.Equal(t, 0, len(rule.ByIPAddress))
	}

	{ // lowercase
		{ // second
			rule, err := testFunc("--rate-limit-api=10-s")
			require.NoError(t, err)
			require.Equal(t, time.Second, rule.Default.Period)
			require.Equal(t, int64(10), rule.Default.Limit)
			require.Equal(t, 0, len(rule.ByIPAddress))
		}
		{ // minute
			rule, err := testFunc("--rate-limit-api=10-m")
			require.NoError(t, err)
			require.Equal(t, time.Minute, rule.Default.Period)
			require.Equal(t, int64(10), rule.Default.Limit)
			require.Equal(t, 0, len(rule.ByIPAddress))
		}
		{ // hour
			rule, err := testFunc("--rate-limit-api=10-h")
			require.NoError(t, err)
			require.Equal(t, time.Hour, rule.Default.Period)
			require.Equal(t, int64(10), rule.Default.Limit)
			require.Equal(t, 0, len(rule.ByIPAddress))
		}
	}
}

func TestParseGenesisOption(t *testing.T) {
	expectedGenesisKP := keypair.Random()
	expectedCommonKP := keypair.Random()

	{ // empty
		_, _, _, err := parseGenesisOptionFromCSV("")
		require.Equal(t, errors.InvalidGenesisOption, err)
	}

	{ // empty ,
		_, _, _, err := parseGenesisOptionFromCSV("  ,  ,  ,")
		require.Equal(t, errors.InvalidGenesisOption, err)
	}

	{ // only genesis
		_, _, _, err := parseGenesisOptionFromCSV(fmt.Sprintf("%s", expectedGenesisKP.Address()))
		require.Equal(t, errors.InvalidGenesisOption, err)
	}

	{ // genesis,common
		genesisKP, commonKP, balance, err := parseGenesisOptionFromCSV(
			fmt.Sprintf("%s,%s", expectedGenesisKP.Address(), expectedCommonKP.Address()),
		)
		require.NoError(t, err)
		require.Equal(t, expectedGenesisKP.Address(), genesisKP.Address())
		require.Equal(t, expectedCommonKP.Address(), commonKP.Address())
		require.Equal(t, common.MaximumBalance, balance)
	}

	{ // genesis,common,balance
		expectedBalance := common.Amount(33333333)
		genesisKP, commonKP, balance, err := parseGenesisOption(
			expectedGenesisKP.Address(), expectedCommonKP.Address(), expectedBalance.String(),
		)
		require.NoError(t, err)
		require.Equal(t, expectedGenesisKP.Address(), genesisKP.Address())
		require.Equal(t, expectedCommonKP.Address(), commonKP.Address())
		require.Equal(t, expectedBalance, balance)
	}

	{ // genesis,common,balance, but invalid balance
		_, _, _, err := parseGenesisOption(
			expectedGenesisKP.Address(), expectedCommonKP.Address(), "a33333333",
		)
		require.Error(t, err)
	}

	{ // genesis,common,balance, but invalid genesis key
		_, _, _, err := parseGenesisOption(expectedGenesisKP.Address()[:3], expectedCommonKP.Address(), "10")
		require.Equal(t, errors.NotPublicKey.Code, err.(*errors.Error).Code)
	}

	{ // genesis,common,balance, but invalid common key
		_, _, _, err := parseGenesisOption(expectedGenesisKP.Address(), expectedCommonKP.Address()[:3], "10")
		require.Equal(t, errors.NotPublicKey.Code, err.(*errors.Error).Code)
	}
}
