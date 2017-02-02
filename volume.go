package nfs

import (
	"os"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/xdr"
)

type Volume struct {
	*rpc.Client

	auth    rpc.Auth
	fh      []byte
	dirPath string

	uid, gid uint32
}

type MkdirArgs struct {
	rpc.Header
	Where Diropargs3
	Attrs Sattr3
}

func (v *Volume) Mkdir(path string, perm os.FileMode) error {
	buf, err := v.Call(&MkdirArgs{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    NFS3_PROG,
			Vers:    NFS3_VERS,
			Proc:    NFSPROC3_MKDIR,
			Cred:    v.auth,
			Verf:    rpc.AUTH_NULL,
		},
		Where: Diropargs3{
			FH:       v.fh,
			Filename: path,
		},
		Attrs: Sattr3{
			Mode: SetMode{
				Set:  uint32(1),
				Mode: uint32(perm.Perm()),
			},
			UID: SetUID{
				Set: uint32(1),
				UID: v.uid,
			},
			GID: SetUID{
				Set: uint32(1),
				UID: v.gid,
			},
		},
	})

	if err != nil {
		return err
	}

	res, buf := xdr.Uint32(buf)
	switch res {
	case NFS3_OK:
		return nil

	default:
		return NFS3Error(res)
	}

	return nil
}

func NewTarget(nt, addr string, auth rpc.Auth, fh []byte, dirpath string, uid, gid uint32) (*Volume, error) {
	m := rpc.Mapping{
		Prog: NFS3_PROG,
		Vers: NFS3_VERS,
		Prot: rpc.IPPROTO_TCP,
		Port: 0,
	}

	client, err := DialService(nt, addr, m)
	if err != nil {
		return nil, err
	}

	vol := &Volume{
		Client:  client,
		auth:    auth,
		fh:      fh,
		dirPath: dirpath,
		uid:     uid,
		gid:     gid,
	}

	return vol, nil
}
