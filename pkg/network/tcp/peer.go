package tcp

import (
	"boscoin.io/sebak/pkg/network"
	"boscoin.io/sebak/pkg/wire/message"
	"errors"
	"fmt"
	"net"
	"sync"
)

type peer struct {
	conn net.Conn

	pubKey message.PeerId

	inbound bool
}

func newPeer(inbound bool, conn net.Conn, hello *message.HelloMessage) *peer {
	return &peer{
		conn:    conn,
		pubKey:  hello.PubKey,
		inbound: inbound,
	}
}

func (o *peer) Id() message.PeerId {
	return o.pubKey
}

func (o *peer) IsInbound() bool {
	return o.inbound
}

func (o *peer) RemoteAddr() string {
	return o.conn.RemoteAddr().String()
}

func (o *peer) Write(msg []byte) (n int, err error) {
	return o.conn.Write(msg)
}

func (o *peer) Close() error {
	return o.conn.Close()
}

type peerstore struct {
	lock sync.Mutex

	peers map[message.PeerId]network.Peer
}

func newPeerstore() network.Peerstore {
	return &peerstore{
		peers: make(map[message.PeerId]network.Peer),
	}
}

func (o *peerstore) Get(id message.PeerId) (network.Peer, error) {
	if p, ok := o.peers[id]; ok {
		return p, nil
	} else {
		return nil, errors.New("no peer found")
	}
}

func (o *peerstore) Put(id message.PeerId, peer network.Peer) error {
	o.lock.Lock()
	defer o.lock.Unlock()

	if _, ok := o.peers[id]; ok {
		return errors.New(fmt.Sprintf("PeerId %s already exists", id.Abbr(6)))
	} else {
		o.peers[id] = peer
		return nil
	}
}

func (o *peerstore) List() []network.Peer {
	values := make([]network.Peer, 0)

	for _, peer := range o.peers {
		values = append(values, peer)
	}

	return values
}
