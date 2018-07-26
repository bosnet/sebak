package node

import (
	"fmt"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestNode(t *testing.T) {
	kp1, _ := keypair.Random()
	kp2, _ := keypair.Random()
	kp3, _ := keypair.Random()

	knownPeers := []*url.URL{
		parseURL(fmt.Sprintf("tcp://%s@127.0.0.1:10001", kp1.Address())),
		parseURL(fmt.Sprintf("tcp://%s@127.0.0.1:10002", kp2.Address())),
		parseURL(fmt.Sprintf("tcp://%s@127.0.0.1:10003", kp3.Address())),
	}
	node1 := newNode("tcp://0.0.0.0:10001", kp1, knownPeers)
	node2 := newNode("tcp://0.0.0.0:10002", kp2, knownPeers)
	node3 := newNode("tcp://0.0.0.0:10003", kp3, knownPeers)

	node1.start()
	node2.start()
	node3.start()

	defer func() {
		os.RemoveAll(node1.config.DataDir)
		os.RemoveAll(node2.config.DataDir)
		os.RemoveAll(node3.config.DataDir)
	}()

	timer := time.NewTimer(time.Second * 3)
	<-timer.C

	assert.Equal(t, 2, len(node1.server.networkManager.Peers().List()))
	assert.Equal(t, 2, len(node2.server.networkManager.Peers().List()))
	assert.Equal(t, 2, len(node3.server.networkManager.Peers().List()))
}

func newNode(listenAddress string, kp *keypair.Full, validators []*url.URL) *Node {
	var config Config

	config.ListenAddresses = []*url.URL{parseURL(listenAddress)}
	config.KeyPair = kp
	config.Validators = validators
	config.DataDir, _ = ioutil.TempDir("/tmp", "sebak-ldb")

	return NewNode(&config)
}

func parseURL(rawUrl string) *url.URL {
	u, _ := url.Parse(rawUrl)
	return u
}
