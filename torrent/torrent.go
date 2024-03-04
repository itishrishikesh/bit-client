package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/jackpal/bencode-go"
)

type Info struct {
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
}

type Torrent struct {
	Announce  string   `bencode:"announce"`
	Comment   string   `bencode:"comment"`
	CreatedBy string   `bencode:"created by"`
	Info      Info     `bencode:"info"`
	UrlList   []string `bencode:"url-list"`
}

type TorrentFile struct {
	Announce     string
	InfoHash     [20]byte
	PiecesHash   [][20]byte
	PiecesLength int
	Length       int
	Name         string
}

// ToTorrentFile convert Torrent to TorrentFile struct
func (t *Torrent) ToTorrenFile() (*TorrentFile, error) {
	b, err := t.Info.splitPieceHashes()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = bencode.Marshal(&buf, t.Info)
	if err != nil {
		return nil, err
	}

	return &TorrentFile{
		Announce:     t.Announce,
		InfoHash:     sha1.Sum(buf.Bytes()),
		PiecesHash:   b,
		Length:       t.Info.Length,
		Name:         t.Info.Name,
		PiecesLength: t.Info.PieceLength,
	}, nil
}

func (i Info) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

// Read reads torrent file and returns Torrent instance.
func Read(r io.Reader) (*Torrent, error) {
	torrent := Torrent{}
	err := bencode.Unmarshal(r, &torrent)
	if err != nil {
		return nil, err
	}
	return &torrent, nil
}

// TrackerUrl Builds and Returns a tracker url based id, and port.
func (t *TorrentFile) TrackerUrl(peerId [20]byte, port uint16) (string, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return "", err
	}
	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(peerId[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}
	base.RawQuery = params.Encode()
	return base.String(), nil
}
