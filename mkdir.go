package nfs

import (
	"os"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/util"
	"github.com/davecheney/nfs/xdr"
)

func (v *Target) call(c interface{}) error {
	buf, err := v.Call(c)

	res, buf := xdr.Uint32(buf)
	switch res {
	case NFS3_OK:
		return nil

	default:
		err = NFS3Error(res)
		return err
	}

}

func (v *Target) Mkdir(path string, perm os.FileMode) error {
	type MkdirArgs struct {
		rpc.Header
		Where Diropargs3
		Attrs Sattr3
	}

	err := v.call(&MkdirArgs{
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
		},
	})

	if err != nil {
		util.Debugf("mkdir(%s): %s", path, err.Error())
		return err
	}

	util.Debugf("mkdir(%s): created successfully", path)
	return nil
}

func (v *Target) RmDir(path string) error {
	type RmDir3Args struct {
		rpc.Header
		Object Diropargs3
	}

	err := v.call(&RmDir3Args{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    NFS3_PROG,
			Vers:    NFS3_VERS,
			Proc:    NFSPROC3_RMDIR,
			Cred:    v.auth,
			Verf:    rpc.AUTH_NULL,
		},
		Object: Diropargs3{
			FH:       v.fh,
			Filename: path,
		},
	})

	if err != nil {
		util.Debugf("rmdir(%s): %s", path, err.Error())
		return err
	}

	util.Debugf("rmdir(%s): deleted successfully", path)
	return nil
}
