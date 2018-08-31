package sebaknetwork

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"unicode"

	"math/rand"
	"net"
	"strconv"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
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

func ExampleHttp2NetworkConfigCreateWithNonTLS() {

	var config HTTP2NetworkConfig
	endpoint, err := sebakcommon.NewEndpointFromString("https://localhost:5000?NodeName=n1")
	if err != nil {
		fmt.Print("Error in NewEndpointFromString")
	}
	queries := endpoint.Query()
	queries.Add("TLSCertFile", "")
	queries.Add("TLSKeyFile", "")
	endpoint.RawQuery = queries.Encode()

	config, err = NewHTTP2NetworkConfigFromEndpoint(endpoint)
	if err != nil {
		fmt.Print("Error in NewHTTP2NetworkConfigFromEndpoint")
	}
	fmt.Println(config.NodeName)
	fmt.Println(config.Addr)

	// Output: n1
	// localhost:5000
}

const (
	dirPath  = "tmp"
	certPath = "cert.pem"
	keyPath  = "key.pem"
)

// Waiting until the server is ready
func pingAndWait(t *testing.T, c0 NetworkClient) {
	waitCount := 0
	for {
		if b, err := c0.GetNodeInfo(); len(b) != 0 && err == nil {
			break
		} else {
			time.Sleep(time.Millisecond * 100)
			waitCount++
			if waitCount > 100 {
				t.Error("Server is not available")
			}
		}
	}
}

func createNewHTTP2Network(t *testing.T) (kp *keypair.Full, mn *HTTP2Network, localNode *sebaknode.LocalNode) {
	g := NewKeyGenerator(dirPath, certPath, keyPath)

	var config HTTP2NetworkConfig
	endpoint, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:%s?NodeName=n1", getPort()))
	if err != nil {
		t.Error(err)
		return
	}

	queries := endpoint.Query()
	queries.Add("TLSCertFile", g.GetCertPath())
	queries.Add("TLSKeyFile", g.GetKeyPath())
	endpoint.RawQuery = queries.Encode()

	config, err = NewHTTP2NetworkConfigFromEndpoint(endpoint)
	if err != nil {
		t.Error(err)
		return
	}
	mn = NewHTTP2Network(config)

	kp, _ = keypair.Random()
	localNode, _ = sebaknode.NewLocalNode(kp, mn.Endpoint(), "")

	mn.SetContext(context.WithValue(context.Background(), "localNode", localNode))

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
	_, s0, localNode := createNewHTTP2Network(t)
	s0.SetMessageBroker(TestMessageBroker{})
	s0.Ready()

	go s0.Start()
	defer s0.Stop()

	c0 := s0.GetClient(s0.Endpoint())
	pingAndWait(t, c0)

	b, err := c0.GetNodeInfo()
	if err != nil {
		t.Error(err)
		return
	}
	v, err := sebaknode.NewValidatorFromString(b)
	if err != nil {
		t.Error(err)
		return
	}

	server := localNode.Endpoint().String()
	client := v.Endpoint().String()

	require.Equal(t, server, client, "Server endpoint and received endpoint should be the same.")
	require.Equal(t, localNode.Address(), v.Address(), "Server address and received address should be the same.")
}

type StringResponseMessageBroker struct {
	msg string
}

func (r StringResponseMessageBroker) ResponseMessage(w http.ResponseWriter, _ string) {
	fmt.Fprintf(w, r.msg)
}

func (r StringResponseMessageBroker) ReceiveMessage(*HTTP2Network, Message) {}

func TestHTTP2NetworkMessageBrokerResponseMessage(t *testing.T) {
	_, s0, localNode := createNewHTTP2Network(t)
	s0.SetMessageBroker(StringResponseMessageBroker{"ResponseMessage"})
	s0.Ready()

	go s0.Start()
	defer s0.Stop()

	c0 := s0.GetClient(s0.Endpoint())
	pingAndWait(t, c0)

	returnMsg, _ := c0.Connect(localNode)

	require.Equal(t, string(returnMsg), "ResponseMessage", "The connectNode and the return should be the same.")
}

func TestHTTP2NetworkConnect(t *testing.T) {
	_, s0, localNode := createNewHTTP2Network(t)
	s0.SetMessageBroker(TestMessageBroker{})
	s0.Ready()

	go s0.Start()
	defer s0.Stop()

	c0 := s0.GetClient(s0.Endpoint())
	pingAndWait(t, c0)

	o, _ := localNode.Serialize()
	nodeStr := removeWhiteSpaces(string(o))

	returnMsg, _ := c0.Connect(localNode)
	returnStr := removeWhiteSpaces(string(returnMsg))

	require.Equal(t, returnStr, nodeStr, "The connectNode and the return should be the same.")
}

func TestHTTP2NetworkSendMessage(t *testing.T) {
	_, s0, _ := createNewHTTP2Network(t)
	s0.SetMessageBroker(TestMessageBroker{})
	s0.Ready()

	go s0.Start()
	defer s0.Stop()

	c0 := s0.GetClient(s0.Endpoint())
	pingAndWait(t, c0)

	msg := NewDummyMessage("findme")
	returnMsg, _ := c0.SendMessage(msg)

	returnStr := removeWhiteSpaces(string(returnMsg))
	sendMsg := removeWhiteSpaces(msg.String())

	require.Equal(t, returnStr, sendMsg, "The sendMessage and the return should be the same.")
}
