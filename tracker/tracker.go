package tracker

import (
	"bit-client/peer"
	"log"
	"net/http"

	"github.com/jackpal/bencode-go"
)

type TrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func Announce(url string) []peer.Peer {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln("unable to get peers from tracker.", err)
	}
	var response TrackerResponse
	err = bencode.Unmarshal(resp.Body, &response)
	if err != nil {
		log.Fatalln("unable to parse tracker response.", err)
	}
	peers, err := peer.Unmarshal([]byte(response.Peers))
	if err != nil {
		log.Fatalln("unable to get peers", err)
	}
	return peers
}
