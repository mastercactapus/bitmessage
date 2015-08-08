package bitmessage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"sort"
	"sync"
	"time"
)

type Node struct {
	port        uint16
	nonce       uint64
	l           net.Listener
	pool        map[uint64]*connection
	poolmx      *sync.RWMutex
	s           Store
	objectIndex map[InvVector]bool
}
type connection struct {
	outgoing bool
	log      *log.Entry
	c        net.Conn
	r        MessageReader
	w        MessageWriter
	node     *Node
	nonce    uint64
	version  *VersionMessage
	inbound  chan Message
	outbound chan Message
}

func GCStoreLoop(s Store) {
	t := time.NewTicker(time.Minute)
	var err error
	for {
		err = gcStore(s)
		if err != nil {
			log.Errorln("GC of database failed:", err)
		}
		<-t.C
	}
}

// gcStore will garbage-collect Store (removing expired objects)
func gcStore(s Store) error {
	objs, err := s.ListObjects()
	if err != nil {
		return err
	}
	var data []byte
	for _, obj := range objs {
		data, err = s.GetObject(obj)
		if err != nil {
			return err
		}
		if time.Now().Unix() >= int64(order.Uint64(data[8:])) {
			log.Infoln("GC:", hex.EncodeToString(obj[:]))
			err = s.DeleteObject(obj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func nonce() uint64 {
	buf := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(err)
	}
	return order.Uint64(buf)
}

func NewNode(lAddr string, s Store) (*Node, error) {
	if lAddr == "" {
		lAddr = ":8444"
	}
	l, err := net.Listen("tcp", lAddr)
	if err != nil {
		return nil, err
	}

	addr := l.Addr().(*net.TCPAddr)

	n := &Node{
		port:        uint16(addr.Port),
		nonce:       nonce(),
		l:           l,
		pool:        make(map[uint64]*connection, 20),
		poolmx:      new(sync.RWMutex),
		s:           s,
		objectIndex: make(map[InvVector]bool, 50000),
	}

	v, err := s.ListObjects()
	if err != nil {
		return nil, err
	}
	for i := range v {
		n.objectIndex[v[i]] = true
	}

	return n, nil
}

func (n *Node) addConnection(c *connection) {
	n.poolmx.Lock()
	n.pool[c.nonce] = c
	n.poolmx.Unlock()
}
func (n *Node) remConnection(c *connection) {
	n.poolmx.Lock()
	delete(n.pool, c.nonce)
	n.poolmx.Unlock()
}

func (n *Node) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	go n.handle(true, conn)
	return nil
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

func newConnection(n *Node, c net.Conn) *connection {
	return &connection{
		log:      log.WithField("RemoteAddr", c.RemoteAddr().String()),
		c:        c,
		r:        MessageReader{c},
		w:        MessageWriter{c},
		node:     n,
		inbound:  make(chan Message, 5),
		outbound: make(chan Message, 5),
	}
}
func (c *connection) readloop() {
	for {
		c.c.SetReadDeadline(time.Now().Add(ConnectionTimeout))
		m, err := c.r.ReadMessage()
		if err != nil {
			close(c.inbound)
			c.log.Warnln("error reading message:", err)
			return
		}
		if _, ok := m.(*RawMessage); ok {
			c.log.Warnln("ignoring unknown type:", m.Command())
			continue
		}
		c.inbound <- m
	}
}

func (c *connection) handshake(outgoing bool) error {
	myVers := NewVersionMessage(c.node.nonce, c.node.port)
	sendVersion := func() error {
		_, err := c.w.WriteMessage(myVers)
		if err != nil {
			return fmt.Errorf("failed to send initial message: %s", err.Error())
		}
		m, err := c.r.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read verack in handshake:", err.Error())
		}
		if m.Command() != MessageTypeVerAck {
			return fmt.Errorf("expected verack but got:", m.Command())
		}
		return nil
	}
	var err error
	if outgoing {
		err = sendVersion()
		if err != nil {
			return err
		}
	}
	m, err := c.r.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read remote version:", err.Error())
	}
	if m.Command() != MessageTypeVersion {
		return fmt.Errorf("unexpected message type during handshake (expected 'version'): %s", m.Command())
	}
	v := m.(*VersionMessage)
	if v.Nonce == c.node.nonce {
		return fmt.Errorf("nonce matched our own, terminating self-connection")
	}
	c.nonce = v.Nonce
	c.log = c.log.WithField("UserAgent", v.UserAgent)
	if v.Version < Version {
		return fmt.Errorf("version was %d, less than ours so terminating connection", v.Version)
	}
	if len(v.StreamNumbers) != 1 || v.StreamNumbers[0] != 1 {
		return fmt.Errorf("we are only interested in stream 1, terminating")
	}
	if !v.Services.NodeNetwork {
		return fmt.Errorf("not a normal node, terminating")
	}
	c.version = v
	_, err = c.w.WriteMessage(&VerAckMessage{})
	if err != nil {
		return err
	}
	if !outgoing {
		err = sendVersion()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *connection) serveMessage(m Message) error {
	switch v := m.(type) {
	case *AddrMessage:
		for _, addr := range v.Addresses {
			c.log.Infoln("Got Address:", addr.IP.String())
		}
	case *InvMessage:
		missing := make([]InvVector, 0, len(v.Inventory))
		for _, i := range v.Inventory {
			if !c.node.objectIndex[i] {
				missing = append(missing, i)
			}
		}
		// request in byte-order
		sort.Sort(InvVectors(missing))
		if len(missing) > 0 {
			c.log.Infof("requesting %d missing objects", len(missing))
			c.outbound <- &GetDataMessage{Inventory: missing}
		}
	case *ObjectMessage:
		data, err := v.MarshalBinary()
		if err != nil {
			return err
		}
		vect := CalcVector(data)
		c.log.Infoln("Store:", hex.EncodeToString(vect[:]))
		return c.node.s.SaveObject(vect, data)
	}
	return nil
}

func (n *Node) handle(outgoing bool, conn net.Conn) {
	c := newConnection(n, conn)
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
	err := c.handshake(outgoing)
	if err != nil {
		c.log.Warnln(err)
		return
	}

	n.addConnection(c)
	// TODO: send addrs

	im := &InvMessage{}
	im.Inventory, err = c.node.s.ListObjects()
	c.outbound <- im
	im = nil

	go c.readloop()
	var m Message
	for {
		select {
		case m = <-c.inbound:
			c.log.Infoln("recv:", m.Command())
			err = c.serveMessage(m)
			if err != nil {
				c.log.Warnln(err)
				return
			}
		case m = <-c.outbound:
			c.log.Infoln("send:", m.Command())
			_, err = c.w.WriteMessage(m)
			if err != nil {
				c.log.Warnln("send message failed:", err)
				return
			}
		}
	}
}
