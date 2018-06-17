package sebaknetwork

import (
	"crypto/ecdsa"
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

var (
	host     = "localhost"
	validFor = time.Duration(1000000)
	rsaBits  = 2048
)

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

type KeyGenerator struct {
	certPath string
	keyPath  string
}

var (
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

func NewKeyGenerator(endPoint string) *KeyGenerator {
	p := &KeyGenerator{}

	p.certPath = fmt.Sprintf("%s/%s_%s%s", tlsDirPath, tlsPrefix, endPoint, certPostfix)
	p.keyPath = fmt.Sprintf("%s/%s_%s%s", tlsDirPath, tlsPrefix, endPoint, keyPostfix)

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
	var priv interface{}
	var err error
	priv, err = rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Debug("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Debug("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
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

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Debug("Failed to create certificate: %s", err)
	}

	os.Mkdir(dirPath, 0777)

	certOut, err := os.Create(certPath)
	if err != nil {
		log.Debug("failed to open %s for writing: %s", certPath, err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	log.Debug("written " + certPath + "\n")

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Debug("failed to open %s for writing:", keyPath, err)
		return
	}
	pem.Encode(keyOut, pemBlockForKey(priv))
	keyOut.Close()
	log.Debug("written " + keyPath + "\n")
}
