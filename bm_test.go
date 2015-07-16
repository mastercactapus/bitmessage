package bitmessage

import (
	"net"
	"testing"
	"time"
)

func TestVerAck(t *testing.T) {
	c, err := net.Dial("tcp", ":8444")
	if err != nil {
		t.Fatal(err)
	}

	var v VersionMessage
	v.Version = Version
	v.Services.NodeNetwork = true
	v.Timestamp = time.Now()
	v.AddressFrom.Services = v.Services
	v.UserAgent = UserAgent
	v.StreamNumbers = []uint64{1}

	r := &MessageReader{c}
	w := &MessageWriter{c}
	_, err = w.WriteMessage(&v)
	if err != nil {
		t.Fatal("write message failed", err)
	}
	m, err := r.ReadMessage()
	if err != nil {
		t.Fatal("read message failed", err)
	}
	t.Log("okay", m.Command())
	m, err = r.ReadMessage()
	if err != nil {
		t.Fatal("read message failed", err)
	}
	t.Fatal("okay", m.Command())
}
