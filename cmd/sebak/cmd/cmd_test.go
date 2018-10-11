package cmd

import (
	"testing"

	"boscoin.io/sebak/lib/node"
	"github.com/stretchr/testify/require"
)

func TestParseFlagValidators(t *testing.T) {
	vs, err := parseFlagValidators("https://localhost:12346?address=GDPQ2LBYP3RL3O675H2N5IEYM6PRJNUA5QFMKXIHGTKEB5KS5T3KHFA2")
	require.Nil(t, err)
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
	require.NotNil(t, err)
}
