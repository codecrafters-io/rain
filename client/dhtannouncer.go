package client

import (
	"net"

	"github.com/cenkalti/rain/internal/torrent/dht"
	node "github.com/nictuku/dht"
)

type dhtAnnouncer struct {
	node     *node.DHT
	infoHash string
	port     int
	peersC   chan []*net.TCPAddr
}

var _ dht.DHT = (*dhtAnnouncer)(nil)

func newDHTAnnouncer(node *node.DHT, infoHash []byte, port int) *dhtAnnouncer {
	return &dhtAnnouncer{
		node:     node,
		infoHash: string(infoHash),
		port:     port,
		peersC:   make(chan []*net.TCPAddr),
	}
}

func (a *dhtAnnouncer) Announce() {
	a.node.PeersRequestPort(a.infoHash, true, a.port)
}

func (a *dhtAnnouncer) Peers() chan []*net.TCPAddr {
	return a.peersC
}
