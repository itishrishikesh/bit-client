package message

import (
	"encoding/binary"
	"io"
)

type messageId uint8

const (
	MsgChoke         messageId = 0
	MsgUnchoke       messageId = 1
	MsgInterested    messageId = 2
	MsgNotInterested messageId = 3
	MsgHave          messageId = 4
	MsgBitField      messageId = 5
	MsgRequest       messageId = 6
	MsgPiece         messageId = 7
	MsgCancel        messageId = 8
)

type Message struct {
	ID      messageId
	Payload []byte
}

func New(typ messageId) []byte {
	m := Message{ID: typ}
	return m.Serialize()
}

func (m *Message) Serialize() []byte {
	if m == nil {
		return make([]byte, 4)
	}
	length := uint32(len(m.Payload) + 1) // +1 for id
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)
	return buf
}

func ReadMessage(r io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lengthBuf)

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	messageBuf := make([]byte, length)
	_, err = io.ReadFull(r, messageBuf)
	if err != nil {
		return nil, err
	}

	m := Message{
		ID:      messageId(messageBuf[0]),
		Payload: messageBuf[1:],
	}

	return &m, nil
}

func FormatRequest(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: MsgRequest, Payload: payload}
}
