package nfs

import (
	"fmt"
	"math/rand"
	"net"
	"os/user"
	"time"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/util"
)

// NFS version 3
// RFC 1813

const (
	NFS3_PROG = 100003
	NFS3_VERS = 3

	// program methods
	NFSPROC3_LOOKUP      = 3
	NFSPROC3_MKDIR       = 9
	NFSPROC3_RMDIR       = 13
	NFSPROC3_READDIRPLUS = 17

	// file types
	NF3REG  = 1
	NF3DIR  = 2
	NF3BLK  = 3
	NF3CHR  = 4
	NF3LNK  = 5
	NF3SOCK = 6
	NF3FIFO = 7
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

type Fattr struct {
	Type                uint32
	Mode                uint32
	Nlink               uint32
	UID                 uint32
	GUID                uint32
	Size                uint64
	Used                uint64
	SpecData            [2]uint32
	FSID                uint64
	Fileid              uint64
	Atime, Mtime, Ctime NFS3Time
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

	var ldr *net.TCPAddr
	usr, err := user.Current()

	// Unless explicitly configured, the target will likely reject connections
	// from non-privileged ports.
	if err == nil && usr.Uid == "0" {
		r1 := rand.New(rand.NewSource(time.Now().UnixNano()))

		var p int
		for p = r1.Intn(1024); p < 0; {
		}

		util.Debugf("using random port %d", p)
		ldr = &net.TCPAddr{
			Port: p,
		}
	}

	client, err := rpc.DialTCP(nt, ldr, fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return nil, err
	}

	return client, nil
}
