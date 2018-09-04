package sebak

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode"

	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
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

const (
	dirPath  = "tmp"
	certPath = "cert.pem"
	keyPath  = "key.pem"
)

// Waiting until the server is ready
func pingAndWait(t *testing.T, c0 sebaknetwork.NetworkClient) {
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

func createNewHTTP2Network(t *testing.T) (kp *keypair.Full, mn *sebaknetwork.HTTP2Network, nodeRunner *NodeRunner) {
	g := sebaknetwork.NewKeyGenerator(dirPath, certPath, keyPath)

	var config sebaknetwork.HTTP2NetworkConfig
	endpoint, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:%s?NodeName=n1", getPort()))
	if err != nil {
		t.Error(err)
		return
	}

	queries := endpoint.Query()
	queries.Add("TLSCertFile", g.GetCertPath())
	queries.Add("TLSKeyFile", g.GetKeyPath())
	endpoint.RawQuery = queries.Encode()

	config, err = sebaknetwork.NewHTTP2NetworkConfigFromEndpoint(endpoint)
	if err != nil {
		t.Error(err)
		return
	}
	mn = sebaknetwork.NewHTTP2Network(config)

	kp, _ = keypair.Random()
	localNode, _ := sebaknode.NewLocalNode(kp, mn.Endpoint(), "")

	p, _ := NewDefaultVotingThresholdPolicy(30, 30)
	is, _ := NewISAAC(networkID, localNode, p)
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	// Make the latest block
	{
		checkpoint := sebakcommon.MakeGenesisCheckpoint(networkID)
		address := kp.Address()
		balance := BaseFee.MustAdd(BaseFee)
		account := block.NewBlockAccount(address, balance, checkpoint)
		account.Save(st)
		MakeGenesisBlock(st, *account)
	}
	if nodeRunner, err = NewNodeRunner(string(networkID), localNode, p, mn, is, st); err != nil {
		panic(err)
	}

	return
}

type TestMessageBroker struct {
	network *sebaknetwork.HTTP2Network
}

func (r TestMessageBroker) Response(w io.Writer, o []byte) error {
	_, err := w.Write(o)
	return err
}

func (r TestMessageBroker) Receive(sebaknetwork.Message) {}

func removeWhiteSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

func TestHTTP2NetworkGetNodeInfo(t *testing.T) {
	_, s0, nodeRunner := createNewHTTP2Network(t)
	s0.SetMessageBroker(TestMessageBroker{network: s0})
	nodeRunner.Ready()

	go nodeRunner.Start()
	defer nodeRunner.Stop()

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

	server := nodeRunner.Node().Endpoint().String()
	client := v.Endpoint().String()

	require.Equal(t, server, client, "Server endpoint and received endpoint should be the same.")
	require.Equal(t, nodeRunner.Node().Address(), v.Address(), "Server address and received address should be the same.")
}

type StringResponseMessageBroker struct {
	network *sebaknetwork.HTTP2Network
	msg     string
}

func (r StringResponseMessageBroker) Response(w io.Writer, _ []byte) error {
	_, err := w.Write([]byte(r.msg))
	return err
}

func (r StringResponseMessageBroker) Receive(sebaknetwork.Message) {}

func TestHTTP2NetworkMessageBrokerResponseMessage(t *testing.T) {
	_, s0, nodeRunner := createNewHTTP2Network(t)
	s0.SetMessageBroker(StringResponseMessageBroker{network: s0, msg: "ResponseMessage"})
	nodeRunner.Ready()

	go nodeRunner.Start()
	defer nodeRunner.Stop()

	c0 := s0.GetClient(s0.Endpoint())
	pingAndWait(t, c0)

	returnMsg, _ := c0.Connect(nodeRunner.Node())

	require.Equal(t, string(returnMsg), "ResponseMessage", "The connectNode and the return should be the same.")
}

func TestHTTP2NetworkConnect(t *testing.T) {
	_, s0, nodeRunner := createNewHTTP2Network(t)
	s0.SetMessageBroker(TestMessageBroker{network: s0})
	nodeRunner.Ready()

	go nodeRunner.Start()
	defer nodeRunner.Stop()

	c0 := s0.GetClient(s0.Endpoint())
	pingAndWait(t, c0)

	o, _ := nodeRunner.Node().Serialize()
	nodeStr := removeWhiteSpaces(string(o))

	returnMsg, _ := c0.Connect(nodeRunner.Node())
	returnStr := removeWhiteSpaces(string(returnMsg))

	require.Equal(t, returnStr, nodeStr, "The connectNode and the return should be the same.")
}
