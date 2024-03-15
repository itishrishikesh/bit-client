package downloader

import (
	"bit-client/message"
	"bytes"
	"crypto/sha1"
	"fmt"
	"net"
)

type Connection struct {
	Conn      net.Conn
	BitField  message.Bitfield
	PieceHash [][20]byte
}

func (c *Connection) readBitfield() error {
	msg, err := message.ReadMessage(c.Conn)
	if err != nil {
		return err
	}
	c.BitField = msg.Payload
	return nil
}

func (c *Connection) downloadBlocks(index, size int) ([]byte, error) {
	blockSize, pieceSize := 16384, size
	var payload bytes.Buffer
	var i int
	for i = 0; (i + blockSize - 1) < pieceSize; i += (blockSize) {
		c.sendRequest(index, i, 16384)
		msg, err := c.waitForPiece()
		if err != nil {
			return nil, fmt.Errorf("unable to read piece block %d : %v", i, err)
		}
		if msg == nil {
			return nil, fmt.Errorf("unable to read piece - keep alive message recieved")
		}
		payload.Write(msg.Payload[8:])
	}
	if (pieceSize - i) != 0 {
		c.sendRequest(index, i, pieceSize-i)
		msg, err := c.waitForPiece()
		if err != nil {
			return nil, fmt.Errorf("unable to read piece block %d : %v", i, err)
		}
		if msg == nil {
			return nil, fmt.Errorf("unable to read piece - keep alive message recieved")
		}
		payload.Write(msg.Payload[8:])
	}
	hash := sha1.Sum(payload.Bytes())
	if !bytes.Equal(hash[:], c.PieceHash[index][:]) {
		return nil, fmt.Errorf("integrity check failed")
	}
	return payload.Bytes(), nil
}

func (c *Connection) sendUnchokeAndInterested() error {
	_, err := c.Conn.Write(message.New(message.MsgUnchoke))
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(message.New(message.MsgInterested))
	if err != nil {
		return err
	}
	return nil
}

func (c *Connection) waitForUnchoke() error {
	unchoke, err := message.ReadMessage(c.Conn)
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

func (c *Connection) sendRequest(index, begin, length int) error {
	request := message.FormatRequest(index, begin, length)
	_, err := c.Conn.Write(request.Serialize())
	return err
}

func (c *Connection) waitForPiece() (*message.Message, error) {
	msg, err := message.ReadMessage(c.Conn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return c.waitForPiece()
	}
	if msg.ID == message.MsgChoke {
		return nil, fmt.Errorf("peer sent choke")
	}
	if msg.ID != message.MsgPiece {
		return c.waitForPiece()
	}
	return msg, nil
}
