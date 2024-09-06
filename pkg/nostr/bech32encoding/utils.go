package bech32encoding

import (
	"bytes"
)

const (
	TLVDefault byte = iota
	TLVRelay
	TLVAuthor
	TLVKind
)

func readTLVEntry(data []byte) (typ uint8, value []byte) {
	if len(data) < 2 {
		return
	}
	typ = data[0]
	length := int(data[1])
	value = data[2 : 2+length]
	return
}

func writeTLVEntry(buf *bytes.Buffer, typ uint8, value []byte) {
	length := len(value)
	buf.WriteByte(typ)
	buf.WriteByte(uint8(length))
	buf.Write(value)
}
