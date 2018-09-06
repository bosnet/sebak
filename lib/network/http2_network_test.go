package network

import (
	"crypto/tls"
	stderrors "errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
)

func getPort() string {
	const ephemeralStart = 49152
	var testPort = "5000"
	for {
		s := rand.NewSource(int64(time.Now().Nanosecond()))
		r := rand.New(s)
		testPort = strconv.Itoa(r.Intn(65535-ephemeralStart) + ephemeralStart) // ephemeral ports range 49152 ~ 65535

		ln, err := net.Listen("tcp", ":"+testPort)
		if err == nil {
			ln.Close()
			time.Sleep(100 * time.Millisecond)
			break
		}
	}
	return testPort
}

func makeTestHTTP2NetworkForTLS(endpoint *sebakcommon.Endpoint) (network *HTTP2Network, err error) {
	var config HTTP2NetworkConfig
	if config, err = NewHTTP2NetworkConfigFromEndpoint(endpoint); err != nil {
		return
	}

	network = NewHTTP2Network(config)
	go network.Start()

	timer := time.NewTimer(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer func() {
		timer.Stop()
		ticker.Stop()
	}()

	var connected bool
	for _ = range ticker.C {
		if connected {
			break
		}

		select {
		case <-timer.C:
			err = stderrors.New("failed to create HTTP2Network")
			return
		default:
			conn, _ := net.DialTimeout("tcp", net.JoinHostPort("", endpoint.Port()), 500*time.Millisecond)
			if conn != nil {
				conn.Close()
				connected = true
				break
			}
		}
	}

	return network, nil
}

// TestHTTP2NetworkTLSSupport will test the HTTP2Network with TLS support.
func TestHTTP2NetworkTLSSupport(t *testing.T) {
	g := NewKeyGenerator("tls_tmp", "sebak.cert", "sebak.key")
	defer g.Close()

	require.NotNil(t, g)

	queryValues := url.Values{}
	queryValues.Set("NodeName", "showme")
	queryValues.Set("TLSCertFile", g.GetCertPath())
	queryValues.Set("TLSKeyFile", g.GetKeyPath())

	endpoint := &sebakcommon.Endpoint{
		Scheme:   "https",
		Host:     fmt.Sprintf("localhost:%s", getPort()),
		RawQuery: queryValues.Encode(),
	}

	network, err := makeTestHTTP2NetworkForTLS(endpoint)
	require.Nil(t, err)
	defer network.Stop()

	{
		// with normal HTTP2Client
		client, err := sebakcommon.NewHTTP2Client(
			defaultTimeout,
			defaultIdleTimeout,
			false,
		)

		require.Nil(t, err)

		_, err = client.Get(endpoint.String(), http.Header{})
		require.Nil(t, err)
	}

	{
		// with normal HTTPClient
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: transport}

		_, err := client.Get(endpoint.String())
		require.Nil(t, err)
	}
}

// TestHTTP2NetworkWithoutTLS will test the HTTP2Network without TLS support.
// Without TLS configurations, `TLSCertFile`, `TLSKeyFile`, `HTTP2Network`
// will be `HTTP` server, not `HTTPS`.
func TestHTTP2NetworkWithoutTLS(t *testing.T) {
	endpoint, err := sebakcommon.NewEndpointFromString(
		fmt.Sprintf("http://localhost:%s?NodeName=showme", getPort()),
	)
	require.Nil(t, err)

	network, err := makeTestHTTP2NetworkForTLS(endpoint)
	require.Nil(t, err)
	defer network.Stop()

	{
		// with normal HTTP2Client
		client, err := sebakcommon.NewHTTP2Client(
			defaultTimeout,
			defaultIdleTimeout,
			false,
		)
		require.Nil(t, err)

		_, err = client.Get(endpoint.String(), http.Header{})
		require.Nil(t, err)
	}

	{
		// with normal HTTPClient
		_, err := http.Get(endpoint.String())
		require.Nil(t, err)
	}
}
