package peer

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

// Unmarshal converts peers to peer struct
func Unmarshal(bin []byte) ([]Peer, error) {
	const size = 6
	numPeers := len(bin) / size
	if len(bin)%size != 0 {
		err := fmt.Errorf("received malformed peers")
		return nil, err
	}
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * size
		peers[i].IP = net.IP(bin[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(bin[offset+4 : offset+6])
	}
	return peers, nil
}

// ConnectToPeer dials TCP connection to provided peer.
func ConnectToPeer(p Peer) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", p.IP.String()+":"+strconv.Itoa(int(p.Port)), 10*time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
