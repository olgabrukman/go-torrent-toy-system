package rpc

import (
	"bytes"
	"encoding/gob"
	"testing"

	"go-torrent-toy-system/message"
)

func TestEncode(t *testing.T) {
	msg := &message.ChunkRequest{
		FileName: "aow.txt",
		Offset:   80,
		Size:     100,
	}

	buf, _ := Marshal(msg)
	r := bytes.NewReader(buf)

	typ, payload, err := Decode(r)
	if err != nil {
		t.Fatal(err)
	}

	if typ != msg.Type() {
		t.Fatalf("type mismatch: %s != %s", msg.Type(), typ)
	}

	var msg2 message.ChunkRequest

	dec := gob.NewDecoder(bytes.NewReader(payload))

	if err := dec.Decode(&msg2); err != nil {
		t.Fatal(err)
	}

	if *msg != msg2 {
		t.Fatalf("message mismatch: %#v != %#v", *msg, msg2)
	}
}
