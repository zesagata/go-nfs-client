package nfs

import (
	"os"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/util"
	"github.com/davecheney/nfs/xdr"
)

func (v *Target) Mkdir(path string, perm os.FileMode) error {
	type MkdirArgs struct {
		rpc.Header
		Where Diropargs3
		Attrs Sattr3
	}

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
		},
	})

	if err != nil {
		return err
	}

	res, buf := xdr.Uint32(buf)
	switch res {
	case NFS3_OK:
		util.Debugf("mkdir(%s): created successfully", path)
		return nil

	default:
		err = NFS3Error(res)
		util.Debugf("mkdir(%s): %s", path, err.Error())
		return err
	}

	return nil
}

func (v *Target) RmDir(path string) error {
	type RmDir3Args struct {
		rpc.Header
		Object Diropargs3
	}

	buf, err := v.Call(&RmDir3Args{
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
		return err
	}

	res, buf := xdr.Uint32(buf)
	switch res {
	case NFS3_OK:
		util.Debugf("rmdir(%s): deleted successfully", path)
		return nil

	default:
		err = NFS3Error(res)
		util.Debugf("rmdir(%s): %s", path, err.Error())
		return err
	}

	return nil

}
