package rpc

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"hash/crc32"
	"io"

	"go-torrent-toy-system/message"
)

//nolint: gochecknoglobals
var (
	typeLen   = 1
	sizeLen   = binary.Size(uint64(0))
	crcLen    = binary.Size(uint32(0))
	headerLen = typeLen + sizeLen + crcLen
)

/*
Marshal type & payload to wire format

[type:byte][payload size: 8 bytes, uint64][payload crc32: 4 bytes, uint32]
[payload ...]
*/
func Marshal(m message.Message) ([]byte, error) {
	payload, err := gobEncode(m)
	if err != nil {
		return nil, err
	}

	size := uint64(len(payload))
	crc := crc32.ChecksumIEEE(payload)

	buf := make([]byte, headerLen+len(payload))
	buf[0] = byte(m.Type())
	binary.BigEndian.PutUint64(buf[typeLen:], size)

	binary.BigEndian.PutUint32(buf[typeLen+sizeLen:], crc)
	copy(buf[headerLen:], payload)

	return buf, nil
}

func gobEncode(m message.Message) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(m); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decode reads message from r, return type & payload
func Decode(r io.Reader) (message.Type, []byte, error) {
	header := make([]byte, headerLen)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}

	t := message.Type(header[0])
	if t < 1 || t >= message.InvalidType {
		return 0, nil, fmt.Errorf("unknown type - %d", t)
	}

	size := binary.BigEndian.Uint64(header[typeLen : typeLen+sizeLen])
	crc := binary.BigEndian.Uint32(header[typeLen+sizeLen:])

	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}

	if crc32.ChecksumIEEE(payload) != crc {
		return 0, nil, fmt.Errorf("bad CRC")
	}

	return t, payload, nil
}

// UnmarshalPayload load data to struct
func UnmarshalPayload(data []byte, obj interface{}) error {
	buf := bytes.NewBuffer(data)
	return gob.NewDecoder(buf).Decode(obj)
}

// Call does an RPC call
func Call(conn io.ReadWriter, request message.Message, response message.Message) error {
	data, err := Marshal(request)
	if err != nil {
		return err
	}

	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	// No response required
	if response == nil {
		return nil
	}

	typ, payload, err := Decode(conn)
	if err != nil {
		return err
	}

	if typ != response.Type() {
		return fmt.Errorf("response type mismatch: expected %s, got %s", typ, response.Type())
	}

	return UnmarshalPayload(payload, response)
}
