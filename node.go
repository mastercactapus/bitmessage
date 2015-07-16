package bitmessage

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

type Node struct {
	port  uint16
	nonce uint64
	l     net.Listener
}
type connection struct {
	outgoing bool
	log      *log.Entry
	c        net.Conn
	r        MessageReader
	w        MessageWriter
}

func NewNode(bindAddr string) *Node {

}
func (n *Node) Connect(address string) error {

}

func (n *Node) Serve() error {
	var conn net.Conn
	var err error
	for {
		conn, err = n.l.Accept()
		if err != nil {
			return err
		}
		go n.handle(false, conn)
	}
}

func (n *Node) handshake(c *connection) (*VersionMessage, error) {
	myVers := NewVersionMessage(n.nonce, n.port)
	sendVersion := func() error {
		_, err := c.w.WriteMessage(myVers)
		if err != nil {
			return fmt.Errorf("failed to send initial message: %s", err.Error())
		}
		m, err := r.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read verack in handshake:", err.Error())
		}
		if m.Command() != MessageTypeVerAck {
			return fmt.Errorf("expected verack but got:", m.Command())
		}
		return nil
	}
	if outgoing {
		err = sendVersion()
		if err != nil {
			return nil, err
		}
	}
	m, err := c.r.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read remote version:", err.Error())
	}
	if m.Command() != MessageTypeVersion {
		return nil, fmt.Errorf("unexpected message type during handshake (expected 'version'): %s", m.Command())
	}
	v := m.(*VersionMessage)
	if v.Nonce == n.nonce {
		return nil, fmt.Errorf("nonce matched our own, terminating self-connection")
	}
	c.log = c.log.WithField("UserAgent", v.UserAgent)
	if v.Version < Version {
		return nil, fmt.Errorf("version was %d, less than ours so terminating connection", v.Version)
	}
	if len(v.StreamNumbers) != 1 || v.StreamNumbers[0] != 1 {
		return nil, fmt.Errorf("we are only interested in stream 1, terminating")
	}
	if !v.Services.NodeNetwork {
		return nil, fmt.Errorf("not a normal node, terminating")
	}
	w.WriteMessage(&VerAckMessage{})
	if !outgoing {
		err = sendVersion()
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}

func (n *Node) handle(outgoing bool, conn net.Conn) {
	c := &connection{
		outgoing: outgoing,
		log:      l,
		c:        conn,
		r:        MessageReader{conn},
		w:        MessageWriter{conn},
	}
	defer func() {
		err := recover()
		if err != nil {
			c.log.Errorln(err)
		}
		conn.Close()
		c.log.Infoln("connection terminated")
	}()
	c.log.Infoln("new connection")
	conn.SetReadDeadline(time.Now().Add(HandshakeTimeout))
	v, err := n.handshake(c)
	if err != nil {
		c.log.Warnln(err)
		return
	}

	// send addrs

	// register node

	for {
		conn.SetReadDeadline(time.Now().Add(ConnectionTimeout))

	}
}
