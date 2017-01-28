package nfs

import "github.com/davecheney/nfs/rpc"

type MkdirArgs struct {
	rpc.Header
	Where Diropargs3
	Attrs Sattr3
}

type Diropargs3 struct {
	FH       []byte
	Filename string
}

type Sattr3 struct {
	Mode  uint32
	UID   uint32
	GID   uint32
	Size  uint64
	Atime NFS3Time
	Mtime NFS3Time
}

type NFS3Time struct {
	Seconds  uint32
	Nseconds uint32
}

func (v *Volume) Mkdir(name string) error {
	_, err := v.Call(MkdirArgs{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    MOUNT_PROG,
			Vers:    MOUNT_VERS,
			Proc:    MOUNTPROC3_MKDIR,
			Cred:    rpc.AUTH_NULL,
			Verf:    rpc.AUTH_NULL,
		},
		Where: Diropargs3{
			FH:       v.fh,
			Filename: name,
		},
		Attrs: Sattr3{},
	})

	return err
}
