package nfs

import (
	"fmt"
	"math/rand"
	"net"

	"github.com/davecheney/nfs/rpc"
)

// NFS version 3
// RFC 1813

const (
	NFS3_PROG = 100003
	NFS3_VERS = 3

	NFSPROC3_MKDIR = 9
)

type Diropargs3 struct {
	FH       []byte
	Filename string
}

type Sattr3 struct {
	Mode  SetMode
	UID   SetUID
	GID   SetUID
	Size  uint64
	Atime NFS3Time
	Mtime NFS3Time
}

type SetMode struct {
	Set  uint32
	Mode uint32
}

type SetUID struct {
	Set uint32
	UID uint32
}

type NFS3Time struct {
	Seconds  uint32
	Nseconds uint32
}

// Error represents an unexpected I/O behavior.
type Error struct {
	ErrorString string
}

func (err *Error) Error() string { return err.ErrorString }

// Dial an RPC svc after getting the port from the portmapper
func DialService(nt, addr string, prog rpc.Mapping) (*rpc.Client, error) {
	pm, err := rpc.DialPortmapper(nt, addr)
	if err != nil {
		return nil, err
	}
	defer pm.Close()

	port, err := pm.Getport(prog)
	if err != nil {
		return nil, err
	}

	var p int
	for p = rand.Intn(1024); p < 0; {
	}

	ldr := &net.TCPAddr{
		Port: p,
	}

	client, err := rpc.DialTCP(nt, ldr, fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return nil, err
	}

	return client, nil
}
