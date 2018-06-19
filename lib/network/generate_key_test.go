package sebaknetwork

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"github.com/stretchr/testify/assert"
)

func TestGenerateKey(t *testing.T) {
	g := NewKeyGenerator("tls_tmp", "sebak.cert", "sebak.key")
	defer g.Close()

	certPath := "tls_tmp/sebak.cert"
	keyPath := "tls_tmp/sebak.key"

	assert.Equal(t, g.GetCertPath(), certPath)
	assert.Equal(t, g.GetKeyPath(), keyPath)

	assert.Equal(t, sebakcommon.IsExists(certPath), true)
	assert.Equal(t, sebakcommon.IsExists(keyPath), true)

}
