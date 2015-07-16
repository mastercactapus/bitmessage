package bitmessage

const (
	ObjectTypeGetPubKey uint32 = iota
	ObjectTypePubKey
	ObjectTypeMsg
	ObjectTypeBroadcast
)

type ObjectType uint32
