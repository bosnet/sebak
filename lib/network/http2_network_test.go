package sebaknetwork

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"unicode"

	"boscoin.io/sebak/lib/common"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
)

type NonTLSInfo struct{}

func (t NonTLSInfo) GetTLSCertInfo(*sebakcommon.Endpoint) (string, error) {
	return "", nil
}

func (t NonTLSInfo) GetTLSKeyInfo(*sebakcommon.Endpoint) (string, error) {
	return "", nil
}

func ExampleHttp2NetworkConfigCreateWithNonTLS() {
	var config HTTP2NetworkConfig
	endpoint, err := sebakcommon.NewEndpointFromString("https://localhost:5000?NodeName=n1")
	if err != nil {
		fmt.Print("Error in NewEndpointFromString")
	}
	config, err = NewHTTP2NetworkConfigFromEndpoint(NonTLSInfo{}, endpoint)
	if err != nil {
		fmt.Print("Error in NewHTTP2NetworkConfigFromEndpoint")
	}
	fmt.Println(config.NodeName)
	fmt.Println(config.Addr)

	// Output: n1
	// localhost:5000
}

var (
	dirPath  = "tmp"
	certPath = "cert.pem"
	keyPath  = "key.pem"
)

type TempTLSInfo struct{}

func (t TempTLSInfo) GetTLSCertInfo(*sebakcommon.Endpoint) (string, error) {
	return dirPath + "/" + certPath, nil
}

func (t TempTLSInfo) GetTLSKeyInfo(*sebakcommon.Endpoint) (string, error) {
	return dirPath + "/" + keyPath, nil
}

func createNewHTTP2Network(t *testing.T) (kp *keypair.Full, mn *HTTP2Network, validator *sebakcommon.Validator) {
	NewKeyGenerator(dirPath, certPath, keyPath)

	var config HTTP2NetworkConfig
	endpoint, err := sebakcommon.NewEndpointFromString("https://localhost:5000?NodeName=n1")
	if err != nil {
		t.Error(err)
		return
	}
	config, err = NewHTTP2NetworkConfigFromEndpoint(TempTLSInfo{}, endpoint)
	if err != nil {
		t.Error(err)
		return
	}
	mn = NewHTTP2Network(config)

	kp, _ = keypair.Random()
	validator, _ = sebakcommon.NewValidator(kp.Address(), mn.Endpoint(), "")
	validator.SetKeypair(kp)

	mn.SetContext(context.WithValue(context.Background(), "currentNode", validator))

	return
}

type TestMessageBroker struct{}

func (r TestMessageBroker) ResponseMessage(w http.ResponseWriter, o string) {
	fmt.Fprintf(w, o)
}

func (r TestMessageBroker) ReceiveMessage(*HTTP2Network, Message) {}

func removeWhiteSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

func TestHTTP2NetworkGetNodeInfo(t *testing.T) {
	_, s0, currentNode := createNewHTTP2Network(t)
	s0.Ready(TestMessageBroker{})

	go s0.Start()
	defer s0.Stop()

	c0 := s0.GetClient(s0.Endpoint())

	b, err := c0.GetNodeInfo()
	if err != nil {
		t.Error(err)
		return
	}
	v, err := sebakcommon.NewValidatorFromString(b)
	if err != nil {
		t.Error(err)
		return
	}

	server := currentNode.Endpoint().String()
	client := v.Endpoint().String()

	assert.Equal(t, server, client, "Server endpoint and received endpoint should be the same.")
	assert.Equal(t, currentNode.Address(), v.Address(), "Server address and received address should be the same.")
}

func TestHTTP2NetworkConnect(t *testing.T) {
	_, s0, currentNode := createNewHTTP2Network(t)
	s0.Ready(TestMessageBroker{})

	go s0.Start()
	defer s0.Stop()

	c0 := s0.GetClient(s0.Endpoint())

	o, _ := currentNode.Serialize()
	nodeStr := removeWhiteSpaces(string(o))

	returnMsg, _ := c0.Connect(currentNode)
	returnStr := removeWhiteSpaces(string(returnMsg))

	assert.Equal(t, returnStr, nodeStr, "The connectNode and the return should be the same.")
}

func TestHTTP2NetworkSendMessage(t *testing.T) {
	_, s0, _ := createNewHTTP2Network(t)
	s0.Ready(TestMessageBroker{})

	go s0.Start()
	defer s0.Stop()

	c0 := s0.GetClient(s0.Endpoint())

	msg := NewDummyMessage("findme")
	returnMsg, _ := c0.SendMessage(msg)

	returnStr := removeWhiteSpaces(string(returnMsg))
	sendMsg := removeWhiteSpaces(msg.String())

	assert.Equal(t, returnStr, sendMsg, "The sendMessage and the return should be the same.")
}

func TestHTTP2NetworkSendBallot(t *testing.T) {
	_, s0, _ := createNewHTTP2Network(t)
	s0.Ready(TestMessageBroker{})
	go s0.Start()
	defer s0.Stop()

	c0 := s0.GetClient(s0.Endpoint())

	msg := NewDummyMessage("findme")
	returnMsg, _ := c0.SendBallot(msg)

	returnStr := removeWhiteSpaces(string(returnMsg))
	sendMsg := removeWhiteSpaces(msg.String())

	assert.Equal(t, returnStr, sendMsg, "The sendBallot and the return should be the same.")
}
