package sebakcommon

import (
	"fmt"
	"testing"
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
			t.Error("must be failed to parse: address must be given")
			return
		}
	}

	// missing scheme
	{
		endpointURL := "//localhost:1234"
		address := "GAWMRHEPMJFTROGBNGIHRR5QEH7E33F7FZNLF6FC5V67BUZI4N2I7BXG"
		_, err := NewValidatorFromURI(fmt.Sprintf("%s?address=%s", endpointURL, address))
		if err == nil {
			t.Error("must be failed to parse: invalid endpoint must be failed")
			return
		}
	}
}
