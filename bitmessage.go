package bitmessage

import (
	"encoding/binary"
	"errors"
	"io"
	"time"
)

const (
	Version           int32 = 3
	UserAgent               = "go 1.5 package github.com/mastercactapus/bitmessage"
	NodeTimeout             = time.Hour * 3
	HandshakeTimeout        = time.Second * 20
	ConnectionTimeout       = time.Minute * 10
)

var order = binary.BigEndian
var ErrTooLong = errors.New("field length was longer than maximum allowed")
var ErrUnknownType = errors.New("unknown type")
var ErrVarIntOverflow = errors.New("varint overflow (larger than 64 bits)")

func UnmarshalBinaryString(b []byte) (string, int, error) {
	l, s := binary.Uvarint(b)
	if s == 0 {
		return "", 0, io.ErrUnexpectedEOF
	}
	if s < 0 {
		return "", 0, ErrVarIntOverflow
	}
	if int(l)+s > len(b) {
		return "", 0, ErrTooLong
	}
	return string(b[l : l+uint64(s)]), int(l) + s, nil
}
func MarshalBinaryString(val string) ([]byte, error) {
	bstr := []byte(val)
	b := make([]byte, 10+len(bstr))
	s := binary.PutUvarint(b, uint64(len(bstr)))
	copy(b[s:], bstr)
	return b[:s+len(bstr)], nil
}
func UnmarshalBinaryIntList(b []byte) ([]uint64, int, error) {
	l, s := binary.Uvarint(b)
	if s == 0 {
		return nil, 0, io.ErrUnexpectedEOF
	}
	if s < 0 {
		return nil, 0, ErrVarIntOverflow
	}
	if int(l)+s > len(b) {
		return nil, 0, ErrTooLong
	}
	vals := make([]uint64, l)
	p := s
	for i := range vals {
		vals[i], s = binary.Uvarint(b[p:])
		if s == 0 {
			return nil, 0, io.ErrUnexpectedEOF
		}
		if s < 0 {
			return nil, 0, ErrVarIntOverflow
		}
		p += s
	}
	return vals, s, nil
}
func MarshalBinaryIntList(vals []uint64) ([]byte, error) {
	b := make([]byte, 10*(len(vals)+1))
	s := binary.PutUvarint(b, uint64(len(vals)))
	for _, v := range vals {
		s += binary.PutUvarint(b[s:], v)
	}
	return b[:s], nil
}
