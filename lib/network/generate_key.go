package sebaknetwork

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	host          = "localhost"
	validForMonth = time.Hour * 24 * 30
	rsaBits       = 4096
)

type KeyGenerator struct {
	certPath string
	keyPath  string
}

const (
	tlsDirPath  = "tls_tmp"
	tlsPrefix   = "sebak"
	certPostfix = ".cert"
	keyPostfix  = ".key"
)

func remove(filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return
	}

	err := os.Remove(filePath)
	if err != nil {
		absFilePath, absErr := filepath.Abs(filePath)
		if absErr != nil {
			log.Error(fmt.Sprintf("failed to get an absolute path(%s)", filePath), "error", absErr)
		}
		log.Error(fmt.Sprintf("failed to remove a file(%s)", absFilePath), "error", err)
	}
}

func NewKeyGenerator(name string) *KeyGenerator {
	p := &KeyGenerator{}

	p.certPath = fmt.Sprintf("%s/%s_%s%s", tlsDirPath, tlsPrefix, name, certPostfix)
	p.keyPath = fmt.Sprintf("%s/%s_%s%s", tlsDirPath, tlsPrefix, name, keyPostfix)

	remove(p.certPath)
	remove(p.keyPath)

	GenerateKey(tlsDirPath, p.certPath, p.keyPath)

	return p
}

func (g *KeyGenerator) GetCertPath() string {
	return g.certPath
}

func (g *KeyGenerator) GetKeyPath() string {
	return g.keyPath
}

func GenerateKey(dirPath string, certPath string, keyPath string) {
	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Debug("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validForMonth)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Debug("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Self-Signed BOScoin Sebak Certificate"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	template.DNSNames = append(template.DNSNames, "localhost")
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Debug("Failed to create certificate: %s", err)
	}

	os.Mkdir(dirPath, 0777)

	certOut, err := os.Create(certPath)
	if err != nil {
		log.Error("failed to open certificate", "certfile", certPath, "error", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Error("failed to open certificate", "keyfile", keyPath, "error", err)
		return
	}
	block := pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	pem.Encode(keyOut, &block)
	keyOut.Close()
}
