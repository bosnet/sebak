package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"unicode"

	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
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
func pingAndWait(t *testing.T, c0 network.NetworkClient) {
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

func createNewHTTP2Network(t *testing.T) (kp *keypair.Full, n *network.HTTP2Network, nodeRunner *NodeRunner) {
	kp = keypair.Random()
	conf := common.NewTestConfig()
	g := network.NewKeyGenerator(dirPath, certPath, keyPath)

	endpoint := common.MustParseEndpoint(fmt.Sprintf("https://localhost:%s?NodeName=n1", getPort()))
	localNode, _ := node.NewLocalNode(kp, endpoint, "")
	localNode.AddValidators(localNode.ConvertToValidator())

	queries := endpoint.Query()
	queries.Add("TLSCertFile", g.GetCertPath())
	queries.Add("TLSKeyFile", g.GetKeyPath())
	endpoint.RawQuery = queries.Encode()

	config, err := network.NewHTTP2NetworkConfigFromEndpoint(localNode.Alias(), endpoint)
	if err != nil {
		t.Error(err)
		return
	}
	n = network.NewHTTP2Network(config)

	p, _ := consensus.NewDefaultVotingThresholdPolicy(30)

	connectionManager := network.NewValidatorConnectionManager(localNode, n, p, conf)

	st := block.InitTestBlockchain()
	is, _ := consensus.NewISAAC(localNode, p, connectionManager, st, conf, nil)
	tp := transaction.NewPool(conf)
	if nodeRunner, err = NewNodeRunner(localNode, p, n, is, st, tp, conf); err != nil {
		panic(err)
	}

	return
}

type TestMessageBroker struct {
	network  *network.HTTP2Network
	Messages []common.NetworkMessage
}

func (r *TestMessageBroker) Response(w io.Writer, o []byte) error {
	_, err := w.Write(o)
	return err
}

func (r *TestMessageBroker) Receive(m common.NetworkMessage) {
	r.Messages = append(r.Messages, m)
}

func removeWhiteSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

type StringResponseMessageBroker struct {
	network *network.HTTP2Network
	msg     string
}

func (r StringResponseMessageBroker) Response(w io.Writer, _ []byte) error {
	_, err := w.Write([]byte(r.msg))
	return err
}

func (r StringResponseMessageBroker) Receive(common.NetworkMessage) {}

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

// TestGetNodeInfoHandler checks `NodeInfoHandler`
func TestGetNodeInfoHandler(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	endpoint := common.MustParseEndpoint("http://localhost:12345")
	localNode, _ := node.NewLocalNode(keypair.Random(), endpoint, "")
	localNode.AddValidators(localNode.ConvertToValidator())
	conf := common.NewTestConfig()
	isaac, _ := consensus.NewISAAC(
		localNode,
		nil,
		network.NewValidatorConnectionManager(localNode, nil, nil, conf),
		st,
		conf,
		nil,
	)

	var config *network.HTTP2NetworkConfig

	config, _ = network.NewHTTP2NetworkConfigFromEndpoint(localNode.Alias(), endpoint)
	nt := network.NewHTTP2Network(config)

	apiHandler := NetworkHandlerNode{storage: st, consensus: isaac, network: nt, localNode: localNode}

	router := mux.NewRouter()
	router.HandleFunc(NodeInfoHandlerPattern, apiHandler.NodeInfoHandler).Methods("GET")

	server := httptest.NewServer(router)
	defer server.Close()

	{ // without setting PublishEndpoint, `endpoint` of response should be requested URL
		u, _ := url.Parse(server.URL)
		u.Path = NodeInfoHandlerPattern

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var received map[string]interface{}
		err = json.Unmarshal(body, &received)
		require.NoError(t, err)

		require.Equal(t, server.URL, received["endpoint"])
	}

	{ // with setting PublishEndpoint, `endpoint` of response should be requested URL
		publishEndpoint := common.MustParseEndpoint("https://9.9.9.9:54321")
		localNode.SetPublishEndpoint(publishEndpoint)

		u, _ := url.Parse(server.URL)
		u.Path = NodeInfoHandlerPattern

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var received map[string]interface{}
		err = json.Unmarshal(body, &received)
		require.NoError(t, err)

		require.Equal(t, publishEndpoint.String(), received["endpoint"])
	}
}
