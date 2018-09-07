package network

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	g := NewKeyGenerator("tls_tmp", "sebak.cert", "sebak.key")
	defer g.Close()

	certPath := "tls_tmp/sebak.cert"
	keyPath := "tls_tmp/sebak.key"

	require.Equal(t, g.GetCertPath(), certPath)
	require.Equal(t, g.GetKeyPath(), keyPath)

	require.Equal(t, common.IsExists(certPath), true)
	require.Equal(t, common.IsExists(keyPath), true)

}
