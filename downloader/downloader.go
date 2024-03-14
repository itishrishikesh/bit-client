package downloader

import (
	"bit-client/handshake"
	"bit-client/message"
	"bit-client/peer"
	"bit-client/torrent"
	"bit-client/tracker"
	"bytes"
	"crypto/sha1"
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

type Connection struct {
	Conn      net.Conn
	BitField  message.Bitfield
	PieceHash [][20]byte
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

func (d *Connection) readBitfield() error {
	msg, err := message.ReadMessage(d.Conn)
	if err != nil {
		return err
	}
	d.BitField = msg.Payload
	return nil
}

func (d *Connection) sendUnchokeAndInterested() error {
	_, err := d.Conn.Write(message.New(message.MsgUnchoke))
	if err != nil {
		return err
	}
	_, err = d.Conn.Write(message.New(message.MsgInterested))
	if err != nil {
		return err
	}
	return nil
}

func (d *Connection) waitForUnchoke() error {
	unchoke, err := message.ReadMessage(d.Conn)
	if err != nil {
		return err
	}
	if unchoke == nil {
		return fmt.Errorf("keep alive message received instead of unchoke")
	}
	if unchoke.ID != message.MsgUnchoke {
		return fmt.Errorf("expected unchoke but received %d", unchoke.ID)
	}
	return nil
}

func (d *Connection) sendRequest(index, begin, length int) error {
	request := message.FormatRequest(index, begin, length)
	_, err := d.Conn.Write(request.Serialize())
	return err
}

func (d *Connection) waitForPiece() (*message.Message, error) {
	msg, err := message.ReadMessage(d.Conn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return d.waitForPiece()
	}
	if msg.ID == message.MsgChoke {
		return nil, fmt.Errorf("peer sent choke")
	}
	if msg.ID != message.MsgPiece {
		return d.waitForPiece()
	}
	return msg, nil
}

func (d *Connection) downloadBlocks(index, size int) ([]byte, error) {
	blockSize, pieceSize := 16384, size
	var payload bytes.Buffer
	var i int
	for i = 0; (i + blockSize - 1) < pieceSize; i += (blockSize) {
		d.sendRequest(index, i, 16384)
		msg, err := d.waitForPiece()
		if err != nil {
			return nil, fmt.Errorf("unable to read piece block %d : %v", i, err)
		}
		if msg == nil {
			return nil, fmt.Errorf("unable to read piece - keep alive message recieved")
		}
		payload.Write(msg.Payload[8:])
	}
	if (pieceSize - i) != 0 {
		d.sendRequest(index, i, pieceSize-i)
		msg, err := d.waitForPiece()
		if err != nil {
			return nil, fmt.Errorf("unable to read piece block %d : %v", i, err)
		}
		if msg == nil {
			return nil, fmt.Errorf("unable to read piece - keep alive message recieved")
		}
		payload.Write(msg.Payload[8:])
	}
	hash := sha1.Sum(payload.Bytes())
	if !bytes.Equal(hash[:], d.PieceHash[index][:]) {
		return nil, fmt.Errorf("integrity check failed")
	}
	return payload.Bytes(), nil
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

func (d *Downloader) CalculateBoundsForPiece(index int) (begin int, end int) {
	begin = index * d.PieceLength
	end = begin + d.PieceLength
	if end > d.Length {
		end = d.Length
	}
	return begin, end
}

func (d *Downloader) findPieceSize(index int) int {
	begin, end := d.CalculateBoundsForPiece(index)
	return end - begin
}
