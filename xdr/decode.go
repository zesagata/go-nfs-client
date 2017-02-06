package xdr

import (
	"encoding/binary"
	"io"

	xdr "github.com/davecgh/go-xdr/xdr2"
)

func Uint32(b []byte) (uint32, []byte) {
	return binary.BigEndian.Uint32(b[0:4]), b[4:]
}

func Opaque(b []byte) ([]byte, []byte) {
	l, b := Uint32(b)
	return b[:l], b[l:]
}

func Uint32List(b []byte) ([]uint32, []byte) {
	l, b := Uint32(b)
	v := make([]uint32, l)
	for i := 0; i < int(l); i++ {
		v[i], b = Uint32(b)
	}
	return v, b
}

func Read(r io.Reader, val interface{}) error {
	_, err := xdr.Unmarshal(r, val)
	return err
}
