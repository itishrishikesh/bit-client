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
)

type Downloader struct {
	Conn        net.Conn
	Peers       []peer.Peer
	PieceHash   [][20]byte
	Infohash    [20]byte
	BitField    message.Bitfield
	PieceLength int
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
	}
}

func (d *Downloader) Download(fileName string) {
	var download bytes.Buffer
	var err error
	curr := 1
	for _, p := range d.Peers {
		for i := curr; i < len(d.PieceHash); i++ {
			curr = i
			d.Conn, err = peer.ConnectToPeer(p)
			if err != nil {
				break
			}
			_, err := handshake.New(d.Infohash).DoHandshake(d.Conn)
			if err != nil {
				continue
			}
			piece, err := d.downloadPiece(i)
			if piece == nil || err != nil {
				i--
				continue
			}
			download.Write(piece)
		}
	}
	os.WriteFile(fileName, download.Bytes(), 0644)
}

func (d *Downloader) readBitfield() error {
	msg, err := message.ReadMessage(d.Conn)
	if err != nil {
		return err
	}
	d.BitField = msg.Payload
	return nil
}

func (d *Downloader) sendUnchokeAndInterested() error {
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

func (d *Downloader) waitForUnchoke() error {
	unchoke, err := message.ReadMessage(d.Conn)
	if unchoke.ID != message.MsgUnchoke {
		return fmt.Errorf("expected unchoke but received %d", unchoke.ID)
	}
	return err
}

func (d *Downloader) sendRequest(index, begin, length int) error {
	request := message.FormatRequest(index, begin, length)
	_, err := d.Conn.Write(request.Serialize())
	return err
}

func (d *Downloader) waitForPiece() (*message.Message, error) {
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

func (d *Downloader) downloadBlocks(index int) ([]byte, error) {
	fmt.Print("\033[H\033[2J")
	fmt.Printf("Trying to download %d piece out of %d\n", index, len(d.PieceHash))
	blockSize, pieceSize := 16384, d.PieceLength
	var payload bytes.Buffer
	var i int
	for i = 0; i < pieceSize; i += (blockSize) {
		d.sendRequest(index, i, 16384)
		msg, err := d.waitForPiece()
		if err != nil || msg == nil {
			return nil, fmt.Errorf("unable to read piece")
		}
		payload.Write(msg.Payload)
	}
	if (i - pieceSize) != 0 {
		d.sendRequest(index, i, i-pieceSize)
		msg, _ := d.waitForPiece()
		payload.Write(msg.Payload)
	}
	// fmt.Printf("Completed downloading %d piece out of %d\n", index, len(d.PieceHash))
	hash := sha1.Sum(payload.Bytes())
	if bytes.Equal(hash[:], payload.Bytes()) {
		return nil, fmt.Errorf("integrity check failed")
	}
	return payload.Bytes(), nil
}

func (d *Downloader) downloadPiece(index int) ([]byte, error) {
	var payload []byte

	err := d.readBitfield()
	if err != nil {
		return nil, err
	}

	err = d.sendUnchokeAndInterested()
	if err != nil {
		return nil, err
	}

	if d.BitField.HasPiece(index) {
		err := d.waitForUnchoke()
		if err != nil {
			return nil, err
		}
		payload, err = d.downloadBlocks(index)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("peer doesn't have piece")
	}
	defer d.Conn.Close()
	return payload, nil
}
