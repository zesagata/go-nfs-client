package rpc

import (
	"bytes"
	"io"
	"math/rand"
	"time"

	"github.com/fdawg4l/nfs/xdr"
)

type transport interface {
	send([]byte) error
	recv() ([]byte, error)
	io.Closer
}

type Header struct {
	Rpcvers uint32
	Prog    uint32
	Vers    uint32
	Proc    uint32
	Cred    Auth
	Verf    Auth
}

type message struct {
	Xid     uint32
	Msgtype uint32
	Body    interface{}
}

type Auth struct {
	Flavor uint32
	Body   []byte
}

var AuthNull Auth

type AuthUnix struct {
	Stamp       uint32
	Machinename string
	Uid         uint32
	Gid         uint32
	GidLen      uint32
	Gids        uint32
}

// Auth converts a into an Auth opaque struct
func (a AuthUnix) Auth() Auth {
	w := new(bytes.Buffer)
	xdr.Write(w, a)
	return Auth{
		1,
		w.Bytes(),
	}
}

func NewAuthUnix(machinename string, uid, gid uint32) *AuthUnix {
	return &AuthUnix{
		Stamp:       rand.New(rand.NewSource(time.Now().UnixNano())).Uint32(),
		Machinename: machinename,
		Uid:         uid,
		Gid:         gid,
		GidLen:      1,
	}
}
