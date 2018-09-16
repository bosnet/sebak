package node

import (
	"fmt"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/common"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestParseValidatorFromURI(t *testing.T) {
	{
		endpointURL := "https://localhost:1234"
		address := "GAWMRHEPMJFTROGBNGIHRR5QEH7E33F7FZNLF6FC5V67BUZI4N2I7BXG"
		alias := "showme"
		v, err := NewValidatorFromURI(fmt.Sprintf("%s?address=%s&alias=%s", endpointURL, address, alias))
		if err != nil {
			t.Error(err)
			return
		}
		if v.Address() != address {
			t.Errorf("failed to parse: address does not match: '%s' != '%s'", v.Address(), address)
			return
		}
		if v.Alias() != alias {
			t.Errorf("failed to parse: alias does not match: '%s' != '%s'", v.Alias(), alias)
			return
		}
		if v.Endpoint().String() != endpointURL {
			t.Errorf("failed to parse: endpoint does not match: '%s' != '%s'", v.Endpoint(), endpointURL)
			return
		}
	}

	// missing alias
	{
		endpointURL := "https://localhost:1234"
		address := "GAWMRHEPMJFTROGBNGIHRR5QEH7E33F7FZNLF6FC5V67BUZI4N2I7BXG"
		v, err := NewValidatorFromURI(fmt.Sprintf("%s?address=%s", endpointURL, address))
		if err != nil {
			t.Error(err)
			return
		}
		if v.Address() != address {
			t.Errorf("failed to parse: address does not match: '%s' != '%s'", v.Address(), address)
			return
		}
		if v.Endpoint().String() != endpointURL {
			t.Errorf("failed to parse: endpoint does not match: '%s' != '%s'", v.Endpoint(), endpointURL)
			return
		}
	}

	// missing address
	{
		endpointURL := "https://localhost:1234"
		_, err := NewValidatorFromURI(fmt.Sprintf("%s", endpointURL))
		if err == nil {
			t.Error("must fail to parse: address must be given")
			return
		}
	}

	// missing scheme
	{
		endpointURL := "//localhost:1234"
		address := "GAWMRHEPMJFTROGBNGIHRR5QEH7E33F7FZNLF6FC5V67BUZI4N2I7BXG"
		_, err := NewValidatorFromURI(fmt.Sprintf("%s?address=%s", endpointURL, address))
		if err == nil {
			t.Error("must fail to parse: invalid endpoint must be failed")
			return
		}
	}
}

func TestValidatorMarshalJSON(t *testing.T) {
	kp, _ := keypair.Random()

	endpoint, err := common.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	require.Equal(t, nil, err)

	validator, _ := NewValidator(kp.Address(), endpoint, "v1")

	tmpByte, err := validator.MarshalJSON()
	require.Equal(t, nil, err)

	jsonStr := `"alias":"%s","endpoint":"https://localhost:%s","state":"%s"`
	require.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "v1", "5000", "NONE")))
}

func TestValidatorNewValidatorFromString(t *testing.T) {
	validator, _ := NewValidatorFromString([]byte(
		`{
			"address":"GATCSN5N6WST3GIJNOF3P55KZTBXG6KUSEFZFHJHV6ZLYNX3OQS2IJTN",
			"alias":"v1",
			"endpoint":"https://localhost:5000",
			"state":"NONE"
		}`,
	))

	require.Equal(t, "v1", validator.Alias())
	require.Equal(t, "https://localhost:5000", validator.Endpoint().String())
	require.Equal(t, StateNONE, validator.State())
}

func TestValidatorUnMarshalJSON(t *testing.T) {
	kp, _ := keypair.Random()

	endpoint, err := common.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	require.Equal(t, nil, err)

	validator, _ := NewValidator(kp.Address(), endpoint, "node")

	validator.UnmarshalJSON([]byte(
		`{
			"address":"GATCSN5N6WST3GIJNOF3P55KZTBXG6KUSEFZFHJHV6ZLYNX3OQS2IJTN",
			"alias":"v1",
			"endpoint":"https://localhost:5000",
			"state":"NONE"
		}`,
	))
	require.Equal(t, nil, err)

	require.Equal(t, "v1", validator.Alias())
	require.Equal(t, "https://localhost:5000", validator.Endpoint().String())
	require.Equal(t, StateNONE, validator.State())
}
