package downloader

import (
	"bit-client/handshake"
	"bit-client/message"
	"bit-client/peer"
	"bit-client/torrent"
	"bit-client/tracker"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

type Downloader struct {
	Conn        net.Conn
	Peers       []peer.Peer
	PieceHash   [][20]byte
	Infohash    [20]byte
	PieceLength int
	Length      int
}

func New(t *torrent.Torrent) *Downloader {
	file, err := t.ToTorrenFile()
	if err != nil {
		log.Fatalln("failed to parse torrent!", err)
	}
	url, err := file.TrackerUrl([20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, 3000)
	if err != nil {
		log.Fatalln("failed to read tracker url!", err)
	}
	return &Downloader{
		Conn:        nil,
		Peers:       tracker.Announce(url),
		PieceHash:   file.PiecesHash,
		Infohash:    file.InfoHash,
		PieceLength: file.PiecesLength,
		Length:      file.Length,
	}
}

func (d *Downloader) Download(fileName string) {
	var download bytes.Buffer
	downloadedPieces := make([][]byte, len(d.PieceHash))
	queue := make(chan int, len(d.PieceHash))
	completed := make(chan int, len(d.PieceHash))
	for i := 0; i < len(d.PieceHash); i++ {
		queue <- i
	}
	go func() {
		for len(completed) < len(d.PieceHash) {
			for index, p := range d.Peers {
				curr := <-queue
				go func(p peer.Peer, index int) {
					c, err := peer.ConnectToPeer(p)
					if err != nil {
						queue <- curr
						return
					}
					c.SetDeadline(time.Now().Add(30 * time.Second))
					_, err = handshake.New(d.Infohash).DoHandshake(c)
					if err != nil {
						queue <- curr
						return
					}
					b, err := d.downloadPiece(curr, &Connection{Conn: c, PieceHash: d.PieceHash})
					if err != nil {
						queue <- curr
						return
					}
					completed <- curr
					downloadedPieces[curr] = b
					c.Write(message.FormatHave(curr).Serialize())
				}(p, index)
			}
		}
	}()
	for len(completed) != len(d.PieceHash) {
		time.Sleep(30 * time.Millisecond)
		fmt.Print("\033[H\033[2J")
		percetage := float64(len(completed)) / float64(len(d.PieceHash))
		fmt.Printf("Downloading --> %0.2f%%", percetage*100)
	}
	for _, b := range downloadedPieces {
		download.Write(b)
	}
	os.WriteFile(fileName, download.Bytes(), 0644)
}

func (d *Downloader) downloadPiece(index int, conn *Connection) ([]byte, error) {
	var payload []byte

	err := conn.readBitfield()
	if err != nil {
		return nil, err
	}

	err = conn.sendUnchokeAndInterested()
	if err != nil {
		return nil, err
	}

	if conn.BitField.HasPiece(index) {
		err := conn.waitForUnchoke()
		if err != nil {
			return nil, err
		}
		payload, err = conn.downloadBlocks(index, d.findPieceSize(index))
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("peer doesn't have piece")
	}
	defer conn.Conn.Close()
	return payload, nil
}

func (d *Downloader) calculateBoundsForPiece(index int) (begin int, end int) {
	begin = index * d.PieceLength
	end = begin + d.PieceLength
	if end > d.Length {
		end = d.Length
	}
	return begin, end
}

func (d *Downloader) findPieceSize(index int) int {
	begin, end := d.calculateBoundsForPiece(index)
	return end - begin
}
