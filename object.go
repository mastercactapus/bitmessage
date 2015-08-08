package bitmessage

import (
	"crypto/sha512"
	"math"
)

const (
	ObjectTypeGetPubKey uint32 = iota
	ObjectTypePubKey
	ObjectTypeMsg
	ObjectTypeBroadcast
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

type InvVectors []InvVector

func (v InvVectors) Len() int {
	return len(v)
}
func (v InvVectors) Swap(a, b int) {
	v[a], v[b] = v[b], v[a]
}
func (v InvVectors) Less(a, b int) bool {
	for i := 0; i < 32; i++ {
		if v[a][i] != v[b][i] {
			return v[a][i] < v[b][i]
		}
	}
	return false
}

func GetPOWValue(data []byte) uint64 {
	nonce := data[:8]
	dataToCheck := data[8:]
	initialHash := sha512.Sum512(dataToCheck)
	h := sha512.New()
	h.Write(nonce)
	h.Write(initialHash[:])
	resultHash := h.Sum(nil)
	h.Reset()
	resultHash = h.Sum(resultHash)
	return order.Uint64(resultHash)
}

func DoPOW(data []byte, target uint64) uint64 {
	var trialValue uint64 = math.MaxUint64
	payload := data[8:]
	var nonce uint64 = 0
	h := sha512.New()
	initialHash := h.Sum(payload)
	b := make([]byte, 8)
	var resHash []byte
	for trialValue > target {
		nonce++
		h.Reset()
		order.PutUint64(b, nonce)
		h.Write(b)
		resHash = h.Sum(initialHash)
		h.Reset()
		resHash = h.Sum(resHash)
		trialValue = order.Uint64(resHash[:])
	}
	return nonce
}
