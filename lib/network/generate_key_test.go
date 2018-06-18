package sebaknetwork

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func isExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestGenerateKey(t *testing.T) {
	g := NewKeyGenerator("localhost_5001")
	certPath := "tls_tmp/sebak_localhost_5001.cert"
	keyPath := "tls_tmp/sebak_localhost_5001.key"

	assert.Equal(t, g.GetCertPath(), certPath)
	assert.Equal(t, g.GetKeyPath(), keyPath)

	assert.Equal(t, isExists(certPath), true)
	assert.Equal(t, isExists(keyPath), true)

}
