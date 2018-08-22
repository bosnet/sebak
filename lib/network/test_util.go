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
	// "github.com/stretchr/testify/assert"
)

const (
	dirPath  = "tmp"
	certPath = "cert.pem"
	keyPath  = "key.pem"
)

func CreateNewHTTP2Network(t *testing.T) (kp *keypair.Full, mn *HTTP2Network, localNode *sebaknode.LocalNode) {
	g := NewKeyGenerator(dirPath, certPath, keyPath)

	var config HTTP2NetworkConfig
	endpoint, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:%s?NodeName=n1", GetPort()))
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

func GetPort() string {
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

func RemoveWhiteSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

type TestMessageBroker struct{}

func (r TestMessageBroker) ResponseMessage(w http.ResponseWriter, o string) {
	fmt.Fprintf(w, o)
}

func (r TestMessageBroker) ReceiveMessage(*HTTP2Network, Message) {}

type StringResponseMessageBroker struct {
	msg string
}

func (r StringResponseMessageBroker) ResponseMessage(w http.ResponseWriter, _ string) {
	fmt.Fprintf(w, r.msg)
}

func (r StringResponseMessageBroker) ReceiveMessage(*HTTP2Network, Message) {}

func PingAndWait(t *testing.T, c0 NetworkClient) {
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
