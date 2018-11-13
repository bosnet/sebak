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
)

func TestParseFlagValidators(t *testing.T) {
	vs, err := parseFlagValidators("https://localhost:12346?address=GDPQ2LBYP3RL3O675H2N5IEYM6PRJNUA5QFMKXIHGTKEB5KS5T3KHFA2")
	require.NoError(t, err)
	require.Equal(t, 1, len(vs))
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
	{ // weired value
		testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

		var fr cmdcommon.ListFlags
		testCmd.Var(&fr, "rate-limit-api", "")

		cmdline := "--rate-limit-api=showme"
		err := testCmd.Parse(strings.Fields(cmdline))
		require.NoError(t, err)

		_, err = parseFlagRateLimit(fr, common.RateLimitAPI)
		require.Error(t, err)
	}

	{ // valid value
		testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

		var fr cmdcommon.ListFlags
		testCmd.Var(&fr, "rate-limit-api", "")

		cmdline := "--rate-limit-api=10-S"
		err := testCmd.Parse(strings.Fields(cmdline))
		require.NoError(t, err)

		rule, err := parseFlagRateLimit(fr, common.RateLimitAPI)
		require.NoError(t, err)
		require.Equal(t, time.Second, rule.Default.Period)
		require.Equal(t, int64(10), rule.Default.Limit)
		require.Equal(t, 0, len(rule.ByIPAddress))
	}

	{ // multiple value, last will be choose.
		testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

		var fr cmdcommon.ListFlags
		testCmd.Var(&fr, "rate-limit-api", "")

		cmdline := "--rate-limit-api=10-S --rate-limit-api=9-M"
		err := testCmd.Parse(strings.Fields(cmdline))
		require.NoError(t, err)

		rule, err := parseFlagRateLimit(fr, common.RateLimitAPI)
		require.NoError(t, err)
		require.Equal(t, time.Minute, rule.Default.Period)
		require.Equal(t, int64(9), rule.Default.Limit)
		require.Equal(t, 0, len(rule.ByIPAddress))
	}

	{ // with ip address, but `common.RateLimitAPI` will be default
		testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

		var fr cmdcommon.ListFlags
		testCmd.Var(&fr, "rate-limit-api", "")

		allowedIP := "1.2.3.4"
		cmdline := fmt.Sprintf("--rate-limit-api=%s=8-S", allowedIP)
		err := testCmd.Parse(strings.Fields(cmdline))
		require.NoError(t, err)

		rule, err := parseFlagRateLimit(fr, common.RateLimitAPI)
		require.NoError(t, err)
		require.Equal(t, common.RateLimitAPI.Period, rule.Default.Period)
		require.Equal(t, common.RateLimitAPI.Limit, rule.Default.Limit)
		require.Equal(t, 1, len(rule.ByIPAddress))
		require.NotNil(t, rule.ByIPAddress[allowedIP])
		require.Equal(t, time.Second, rule.ByIPAddress[allowedIP].Period)
		require.Equal(t, int64(8), rule.ByIPAddress[allowedIP].Limit)
	}

	{ // with ip address and with default
		testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

		var fr cmdcommon.ListFlags
		testCmd.Var(&fr, "rate-limit-api", "")

		allowedIP := "1.2.3.4"
		cmdline := fmt.Sprintf("--rate-limit-api=11-H --rate-limit-api=%s=8-S", allowedIP)
		err := testCmd.Parse(strings.Fields(cmdline))
		require.NoError(t, err)

		rule, err := parseFlagRateLimit(fr, common.RateLimitAPI)
		require.NoError(t, err)
		require.Equal(t, time.Hour, rule.Default.Period)
		require.Equal(t, int64(11), rule.Default.Limit)
		require.Equal(t, 1, len(rule.ByIPAddress))
		require.NotNil(t, rule.ByIPAddress[allowedIP])
		require.Equal(t, time.Second, rule.ByIPAddress[allowedIP].Period)
		require.Equal(t, int64(8), rule.ByIPAddress[allowedIP].Limit)
	}

	{ // unlimit
		testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

		var fr cmdcommon.ListFlags
		testCmd.Var(&fr, "rate-limit-api", "")

		cmdline := "--rate-limit-api=0-S"
		err := testCmd.Parse(strings.Fields(cmdline))
		require.NoError(t, err)

		rule, err := parseFlagRateLimit(fr, common.RateLimitAPI)
		require.NoError(t, err)
		require.Equal(t, time.Second, rule.Default.Period)
		require.Equal(t, int64(0), rule.Default.Limit)
		require.Equal(t, 0, len(rule.ByIPAddress))
	}

	{ // lowercase
		{ // second
			testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

			var fr cmdcommon.ListFlags
			testCmd.Var(&fr, "rate-limit-api", "")

			cmdline := "--rate-limit-api=10-s"
			err := testCmd.Parse(strings.Fields(cmdline))
			require.NoError(t, err)

			rule, err := parseFlagRateLimit(fr, common.RateLimitAPI)
			require.NoError(t, err)
			require.Equal(t, time.Second, rule.Default.Period)
			require.Equal(t, int64(10), rule.Default.Limit)
			require.Equal(t, 0, len(rule.ByIPAddress))
		}
		{ // minute
			testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

			var fr cmdcommon.ListFlags
			testCmd.Var(&fr, "rate-limit-api", "")

			cmdline := "--rate-limit-api=10-m"
			err := testCmd.Parse(strings.Fields(cmdline))
			require.NoError(t, err)

			rule, err := parseFlagRateLimit(fr, common.RateLimitAPI)
			require.NoError(t, err)
			require.Equal(t, time.Minute, rule.Default.Period)
			require.Equal(t, int64(10), rule.Default.Limit)
			require.Equal(t, 0, len(rule.ByIPAddress))
		}
		{ // hour
			testCmd := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

			var fr cmdcommon.ListFlags
			testCmd.Var(&fr, "rate-limit-api", "")

			cmdline := "--rate-limit-api=10-h"
			err := testCmd.Parse(strings.Fields(cmdline))
			require.NoError(t, err)

			rule, err := parseFlagRateLimit(fr, common.RateLimitAPI)
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
