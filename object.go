package bitmessage

import (
	"crypto/sha512"
	"time"
)

const (
	ObjectTypeGetPubKey uint32 = iota
	ObjectTypePubKey
	ObjectTypeMsg
	ObjectTypeBroadcast
)

const (
	trialValueStart uint64 = 99999999999999999999
)

type ObjectType uint32

type GetPubKeyOldObject struct {
	Ripe [20]byte
}
type GetPubKeyObject struct {
	Tag [32]byte
}

type PubKey2Object struct {
}
type PubKey3Object struct {
}
type MsgObject struct {
	Encrypted []byte
}

func GetPOWValue(data []byte) uint64 {
	nonce := data[:8]
	dataToCheck := data[8:]
	initialHash := sha512.Sum512(dataToCheck)
	h := sha512.New()
	h.Write(nonce)
	h.Write(initialHash[:])
	resultHash := h.Sum(nil)
	resultHash = sha512.Sum512(resultHash[:])
	return order.Uint64(resultHash[:])
}

func DoPOW(data []byte, target uint64) uint64 {
	trialValue := trialValueStart
	payload := data[8:]
	var nonce uint64 = 0
	h := sha512.New()
	initialHash := h.Sum(payload)
	b := make([]byte, 8)
	var resHash [64]byte
	for trialValue > target {
		nonce++
		h.Reset()
		order.PutUint64(b, nonce)
		h.Write(p)
		resHash = h.Sum(initialHash)
		h.Reset()
		resHash = h.Sum(resHash)
		trialValue = order.Uint64(resHash[:])
	}
	return nonce
}
