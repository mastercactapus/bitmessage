package bitmessage

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
