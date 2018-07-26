package tcp

import (
	"boscoin.io/sebak/pkg/network"
	"boscoin.io/sebak/pkg/support/logger"
	"boscoin.io/sebak/pkg/wire"
	"boscoin.io/sebak/pkg/wire/message"
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/url"
	"sync"
)

type NetworkManager struct {
	lock   sync.Mutex
	logger *logger.Logger

	receivers []network.Receiver

	params *Params

	protocol wire.Protocol

	peerstore network.Peerstore
}

type Params struct {
	ListenAddresses []*url.URL

	PubKey message.PeerId

	NetworkPassphrase string

	Protocol wire.Protocol
}

func NewTcpNetwork(params *Params) *NetworkManager {
	return &NetworkManager{
		logger:    logger.NewLogger("network"),
		params:    params,
		protocol:  params.Protocol,
		peerstore: newPeerstore(),
		receivers: make([]network.Receiver, 0),
	}
}

func (o *NetworkManager) Start() {
	for _, receiver := range o.receivers {
		receiver.Start()
	}

	go o.listen()
}

func (o *NetworkManager) Stop() {
	for _, receiver := range o.receivers {
		receiver.Stop()
	}
}

func (o *NetworkManager) Connect(address *url.URL) error {
	o.logger.Info("msg", fmt.Sprintf("Connecting to %s://%s", address.Scheme, address.Host))
	if conn, err := net.DialTimeout(address.Scheme, address.Host, connectTimeout); err == nil {
		if p, err := o.handshake(conn, false); err != nil {
			conn.Close()
			o.logger.Warn("msg", "Failed to handshake", "address", address, "error", err)
			return err
		} else if p != nil {
			o.acceptPeer(p)
		}
	} else {
		o.logger.Warn("msg", fmt.Sprintf("Failed to connect to %s", address), "error", err)
		return err
	}
	return nil
}

func (o *NetworkManager) Send(peerId message.PeerId, msg interface{}) error {
	if peer, err := o.Peers().Get(peerId); err != nil {
		return err
	} else {
		if _, err := o.protocol.Pack(peer, msg); err != nil {
			peer.Close()
			return err
		}
		return nil
	}
}

func (o *NetworkManager) Broadcast(msg interface{}) error {
	var b bytes.Buffer
	if _, err := o.protocol.Pack(&b, msg); err != nil {
		return err
	}

	msgBytes := b.Bytes()
	for _, peer := range o.Peers().List() {
		if _, err := peer.Write(msgBytes); err != nil {
			peer.Close()
		}
	}

	return nil
}

func (o *NetworkManager) Peers() network.Peerstore {
	return o.peerstore
}

func (o *NetworkManager) Protocol() wire.Protocol {
	return o.protocol
}

func (o *NetworkManager) AddReceiver(receiver network.Receiver) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if receiver != nil {
		o.receivers = append(o.receivers, receiver)
	}
}

func (o *NetworkManager) listen() {
	for _, listenAddress := range o.params.ListenAddresses {
		listener, err := net.Listen(listenAddress.Scheme, listenAddress.Host)
		if err != nil {
			o.logger.Info("msg", "Can't listen address", "address", listenAddress)
			continue
		}

		o.logger.Info("msg", fmt.Sprintf("Listening on %s", listenAddress.Host))

		go o.acceptLoop(listener)
	}
}

func (o *NetworkManager) acceptLoop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			o.logger.Info("msg", "Can't accept connection", "error", err)
		}

		go o.accept(conn)
	}
}

func (o *NetworkManager) accept(conn net.Conn) {
	if p, err := o.handshake(conn, true); err != nil {
		o.logger.Debug("msg", "Can't accept connection", "error", err)
		conn.Close()
	} else if p != nil {
		o.acceptPeer(p)
	}
}

func (o *NetworkManager) acceptPeer(p *peer) {
	peerId := p.Id()
	if err := o.peerstore.Put(p.Id(), p); err == nil {
		o.logger.Info("msg", fmt.Sprintf("Added a peer from %s", p.RemoteAddr()),
			"from", peerId.Abbr(6), "to", o.params.PubKey.Abbr(6), "inbound", p.IsInbound())

		go o.handleMessages(p)
	} else {
		o.logger.Debug("msg", "Close the duplicate connection")
		p.Close()
	}
}

func (o *NetworkManager) handleMessages(p *peer) {
	for _, receiver := range o.receivers {
		receiver.OnConnect(p.Id())
	}

	for {
		r := bufio.NewReader(p.conn)
		msg, err := o.protocol.Unpack(r)

		if err == nil {
			for _, receiver := range o.receivers {
				receiver.Receive(p.Id(), msg)
			}
		}
	}
}
