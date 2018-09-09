package network

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
)

func TestHTTP2NetworkConfigHTTPSAndTLS(t *testing.T) {
	var nodeName string = "showme"
	{ // HTTPS + TLSCertFile + TLSKeyFile
		queryValues := url.Values{}
		queryValues.Set("TLSCertFile", "faketlscert")
		queryValues.Set("TLSKeyFile", "faketlskey")

		endpoint := &common.Endpoint{
			Scheme:   "https",
			Host:     fmt.Sprintf("localhost:%s", getPort()),
			RawQuery: queryValues.Encode(),
		}

		_, err := NewHTTP2NetworkConfigFromEndpoint(nodeName, endpoint)
		require.Nil(t, err)
	}

	{ // HTTPS + TLSCertFile
		queryValues := url.Values{}
		queryValues.Set("TLSCertFile", "faketlscert")

		endpoint := &common.Endpoint{
			Scheme:   "https",
			Host:     fmt.Sprintf("localhost:%s", getPort()),
			RawQuery: queryValues.Encode(),
		}

		_, err := NewHTTP2NetworkConfigFromEndpoint(nodeName, endpoint)
		require.NotNil(t, err)
	}

	{ // HTTPS + TLSKeyFile
		queryValues := url.Values{}
		queryValues.Set("TLSKeyFile", "faketlskey")

		endpoint := &common.Endpoint{
			Scheme:   "https",
			Host:     fmt.Sprintf("localhost:%s", getPort()),
			RawQuery: queryValues.Encode(),
		}

		_, err := NewHTTP2NetworkConfigFromEndpoint(nodeName, endpoint)
		require.NotNil(t, err)
	}

	{ // HTTP
		queryValues := url.Values{}
		queryValues.Set("TLSCertFile", "faketlscert")
		queryValues.Set("TLSKeyFile", "faketlskey")

		endpoint := &common.Endpoint{
			Scheme:   "http",
			Host:     fmt.Sprintf("localhost:%s", getPort()),
			RawQuery: queryValues.Encode(),
		}

		_, err := NewHTTP2NetworkConfigFromEndpoint(nodeName, endpoint)
		require.Nil(t, err)
	}
}
