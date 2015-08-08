package bitmessage

import (
	"net"
	"time"
)

type Address struct {
	Services VersionServices
	IP       net.IP
	Port     uint16
}

type FullAddress struct {
	Time   time.Time
	Stream uint32
	Address
}

func (m *Address) UnmarshalBinary(b []byte) error {
	m.Services.fromValue(order.Uint64(b))
	m.IP = make([]byte, 16)
	copy(m.IP, b[8:])
	m.Port = order.Uint16(b[24:])
	return nil
}
func (m *Address) MarshalBinary() ([]byte, error) {
	b := make([]byte, 28)
	order.PutUint64(b, uint64(m.Services.value()))
	copy(b[8:], m.IP)
	order.PutUint16(b[24:], m.Port)
	return b, nil
}

func (m *FullAddress) UnmarshalBinary(b []byte) error {
	m.Time = time.Unix(int64(order.Uint64(b)), 0)
	m.Stream = order.Uint32(b[8:])
	return m.Address.UnmarshalBinary(b[12:])
}
func (m *FullAddress) MarshalBinary() ([]byte, error) {
	b := make([]byte, 12, 40)
	order.PutUint64(b, uint64(m.Time.Unix()))
	order.PutUint32(b[8:], m.Stream)
	addr, err := m.Address.MarshalBinary()
	if err != nil {
		return nil, err
	}
	b = append(b, addr...)
	return b, nil
}
