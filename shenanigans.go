package bitmessage

// encodeBitmessageUvarint will encode to the bitmessage varint format
func encodeBitmessageUvarint(b []byte, v uint64) int {
	if v < 0xfd {
		b[0] = byte(v)
		return 1
	}
	if v < 0xffff {
		b[0] = 0xfd
		order.PutUint16(b[1:], uint16(v))
		return 3
	}
	if v < 0xffffffff {
		b[0] = 0xfe
		order.PutUint32(b[1:], uint32(v))
		return 5
	}
	b[0] = 0xff
	order.PutUint64(b[1:], uint64(v))
	return 9
}

// decodeBitmessageUvarint will decode from the bitmessage varint format
func decodeBitmessageUvarint(b []byte) (uint64, int) {
	if b[0] < 0xfd {
		return uint64(b[0]), 1
	}
	if b[0] == 0xfd {
		return uint64(order.Uint16(b[1:])), 3
	}
	if b[0] == 0xfe {
		return uint64(order.Uint32(b[1:])), 5
	}
	return order.Uint64(b[1:]), 9
}
