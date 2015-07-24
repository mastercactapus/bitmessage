package bitmessage

import (
	"bytes"
	"crypto/sha512"
	"encoding"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

const (
	MessageMagic     uint32 = 0xe9beb4d9
	MaxMessageLength        = 1600003
)

const (
	MessageTypeVersion MessageType = "version"
	MessageTypeVerAck              = "verack"
	MessageTypeAddr                = "addr"
	MessageTypeInv                 = "inv"
	MessageTypeGetData             = "getdata"
	MessageTypeObject              = "object"
)

const (
	VersionServicesNodeNetwork = 1
)

type MessageType string
type Message interface {
	Command() MessageType
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}
type MessageWriter struct {
	io.Writer
}
type MessageReader struct {
	io.Reader
}

type VersionServices struct {
	NodeNetwork bool
}
type VersionMessage struct {
	Version       int32
	Services      VersionServices
	Timestamp     time.Time
	AddressRecv   Address
	AddressFrom   Address
	Nonce         uint64
	UserAgent     string
	StreamNumbers []uint64
}
type AddrMessage struct {
	Addresses []FullAddress
}
type InvMessage struct {
	Inventory []InvVector
}
type GetDataMessage struct {
	Inventory []InvVector
}
type ObjectMessage struct {
	Nonce   uint64
	Expires time.Time
	Type    ObjectType
	Version uint64
	Stream  uint64
	Payload []byte
}

type VerAckMessage struct{}
type InvVector [32]byte
type RawMessage struct {
	Type    MessageType
	Payload []byte
}

func (w *MessageWriter) WriteMessage(m Message) (int, error) {
	cmd := []byte(m.Command())
	if len(cmd) > 12 {
		return 0, ErrTooLong
	}
	data, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	b := make([]byte, 24)
	order.PutUint32(b, MessageMagic)
	copy(b[4:], cmd)
	order.PutUint32(b[16:], uint32(len(data)))
	sum := sha512.Sum512(data)
	copy(b[20:], sum[:4])
	n, err := w.Write(b)
	if err != nil {
		return n, err
	}
	n2, err := w.Write(data)
	if err != nil {
		return n + n2, err
	}
	return n + n2, nil
}
func (r *MessageReader) ReadMessage() (Message, error) {
	b := make([]byte, 24)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	if order.Uint32(b) != MessageMagic {
		return nil, fmt.Errorf("Got bad magic value got 0x%s", hex.EncodeToString(b[:4]))
	}
	cmdB := b[4:16]
	padding := bytes.IndexByte(cmdB, 0)
	var cmd MessageType
	if padding < 0 {
		cmd = MessageType(cmdB)
	} else {
		cmd = MessageType(cmdB[:padding])
	}
	l := order.Uint32(b[16:])
	if l > MaxMessageLength {
		return nil, fmt.Errorf("Bad message length %d > max %d", l, MaxMessageLength)
	}
	data := make([]byte, l)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}

	sum := sha512.Sum512(data)
	if !bytes.Equal(sum[:4], b[20:24]) {
		return nil, fmt.Errorf("invalid checksum, expected %s but got %s", hex.EncodeToString(b[20:24]), hex.EncodeToString(sum[:4]))
	}

	var m Message
	switch cmd {
	case MessageTypeVersion:
		m = new(VersionMessage)
	case MessageTypeVerAck:
		m = new(VerAckMessage)
	case MessageTypeAddr:
		m = new(AddrMessage)
	case MessageTypeInv:
		m = new(InvMessage)
	default:
		m = &RawMessage{Type: cmd}
	}
	return m, m.UnmarshalBinary(data)
}

func (s *VersionServices) value() uint64 {
	var v uint64
	if s.NodeNetwork {
		v |= VersionServicesNodeNetwork
	}
	return v
}
func (s *VersionServices) fromValue(value uint64) {
	s.NodeNetwork = value&VersionServicesNodeNetwork != 0
}

func NewVersionMessage(nonce uint64, port uint16) *VersionMessage {
	var v VersionMessage
	v.Version = Version
	v.Services.NodeNetwork = true
	v.Timestamp = time.Now()
	v.AddressFrom.Services = v.Services
	v.AddressFrom.Port = port
	v.UserAgent = UserAgent
	v.Nonce = nonce
	v.StreamNumbers = []uint64{1}
	return &v
}

func (m *VersionMessage) UnmarshalBinary(b []byte) error {
	m.Version = int32(order.Uint32(b))
	m.Services.fromValue(order.Uint64(b[4:]))
	m.Timestamp = time.Unix(int64(order.Uint64(b[12:])), 0)
	err := m.AddressRecv.UnmarshalBinary(b[20:])
	if err != nil {
		return err
	}
	err = m.AddressFrom.UnmarshalBinary(b[46:])
	if err != nil {
		return err
	}
	m.Nonce = order.Uint64(b[72:])
	var n int
	m.UserAgent, n, err = UnmarshalBinaryString(b[80:])
	if err != nil {
		return err
	}
	if n > 5000 {
		return ErrTooLong
	}
	m.StreamNumbers, n, err = UnmarshalBinaryIntList(b[80+n:])
	if err != nil {
		return err
	}
	if len(m.StreamNumbers) > 160000 {
		return ErrTooLong
	}
	return nil
}
func (m *VersionMessage) MarshalBinary() ([]byte, error) {
	b := make([]byte, 80, 1024)
	order.PutUint32(b, uint32(m.Version))
	order.PutUint64(b[4:], uint64(m.Services.value()))
	order.PutUint64(b[12:], uint64(m.Timestamp.Unix()))
	a, err := m.AddressRecv.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(b[20:], a)
	a, err = m.AddressFrom.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(b[46:], a)
	order.PutUint64(b[72:], m.Nonce)
	a, err = MarshalBinaryString(m.UserAgent)
	if err != nil {
		return nil, err
	}
	b = append(b, a...)
	a, err = MarshalBinaryIntList(m.StreamNumbers)
	if err != nil {
		return nil, err
	}
	b = append(b, a...)
	return b, nil
}

func (m *VersionMessage) Command() MessageType {
	return MessageTypeVersion
}

func (m *VerAckMessage) Command() MessageType {
	return MessageTypeVerAck
}
func (m *VerAckMessage) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}
func (m *VerAckMessage) UnmarshalBinary(b []byte) error {
	return nil
}

func (m *RawMessage) Command() MessageType {
	return m.Type
}
func (m *RawMessage) MarshalBinary() ([]byte, error) {
	b := make([]byte, len(m.Payload))
	copy(b, m.Payload)
	return b, nil
}
func (m *RawMessage) UnmarshalBinary(b []byte) error {
	m.Payload = make([]byte, len(b))
	copy(m.Payload, b)
	return nil
}

func (m *AddrMessage) Command() MessageType {
	return MessageTypeAddr
}
func (m *AddrMessage) MarshalBinary() ([]byte, error) {
	n := len(m.Addresses)
	b := make([]byte, 10, n*38+10)
	if n > 1000 {
		n = 1000
	}
	blen := binary.PutUvarint(b, uint64(n))
	b = b[:blen]
	var data []byte
	var err error
	for i := range m.Addresses {
		data, err = m.Addresses[i].MarshalBinary()
		if err != nil {
			return nil, err
		}
		b = append(b, data...)
	}
	return b, nil
}
func (m *AddrMessage) UnmarshalBinary(b []byte) error {
	num, offset := binary.Uvarint(b)
	if num > 1000 {
		return ErrTooLong
	}
	m.Addresses = make([]FullAddress, num)
	var err error
	for i := range m.Addresses {
		err = m.Addresses[i].UnmarshalBinary(b[offset:])
		if err != nil {
			return err
		}
		offset += 38
	}
	return nil
}

func (v InvVector) String() string {
	return hex.EncodeToString(v[:])
}

func (m *InvMessage) Command() MessageType {
	return MessageTypeInv
}
func (m *InvMessage) MarshalBinary() ([]byte, error) {
	b := make([]byte, 10, 10+32*len(m.Inventory))
	blen := binary.PutUvarint(b, uint64(len(m.Inventory)))
	b = b[:blen]
	for i := range m.Inventory {
		b = append(b, m.Inventory[i][:]...)
	}
	return b, nil
}
func (m *InvMessage) UnmarshalBinary(b []byte) error {
	num, offset := binary.Uvarint(b)
	if num*32 > uint64(len(b)) {
		return ErrTooLong
	}
	m.Inventory = make([]InvVector, num)
	for i := range m.Inventory {
		copy(m.Inventory[i][:], b[offset:])
		offset += 32
	}
	return nil
}

func (m *GetDataMessage) Command() MessageType {
	return MessageTypeGetData
}
func (m *GetDataMessage) MarshalBinary() ([]byte, error) {
	b := make([]byte, 10, 10+32*len(m.Inventory))
	blen := binary.PutUvarint(b, uint64(len(m.Inventory)))
	b = b[:blen]
	for i := range m.Inventory {
		b = append(b, m.Inventory[i][:]...)
	}
	return b, nil
}
func (m *GetDataMessage) UnmarshalBinary(b []byte) error {
	num, offset := binary.Uvarint(b)
	if num*32 > uint64(len(b)) {
		return ErrTooLong
	}
	m.Inventory = make([]InvVector, num)
	for i := range m.Inventory {
		copy(m.Inventory[i][:], b[offset:])
		offset += 32
	}
	return nil
}
