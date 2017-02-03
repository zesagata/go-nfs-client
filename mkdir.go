package nfs

import (
	"bytes"
	"os"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/util"
	"github.com/davecheney/nfs/xdr"
)

func (v *Target) call(c interface{}) error {
	buf, err := v.Call(c)
	if err != nil {
		return err
	}

	res, buf := xdr.Uint32(buf)
	if err = NFS3Error(res); err != nil {
		return err
	}

	return nil
}

// Lookup returns a file handle to a given dirent
func (v *Target) Lookup(path string) (*Fattr, []byte, error) {
	type Lookup3Args struct {
		rpc.Header
		What Diropargs3
	}

	buf, err := v.Call(&Lookup3Args{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    NFS3_PROG,
			Vers:    NFS3_VERS,
			Proc:    NFSPROC3_LOOKUP,
			Cred:    v.auth,
			Verf:    rpc.AUTH_NULL,
		},
		What: Diropargs3{
			FH:       v.fh,
			Filename: path,
		},
	})

	if err != nil {
		util.Debugf("lookup(%s): %s", path, err.Error())
		return nil, nil, err
	}

	res, buf := xdr.Uint32(buf)
	if err = NFS3Error(res); err != nil {
		return nil, nil, err
	}

	fh, buf := xdr.Opaque(buf)
	util.Debugf("lookup(%s): FH 0x%x", path, fh)

	var fattrs *Fattr
	attrFollows, buf := xdr.Uint32(buf)
	if attrFollows != 0 {
		r := bytes.NewBuffer(buf)
		fattrs = &Fattr{}
		if err = xdr.Read(r, fattrs); err != nil {
			return nil, nil, err
		}

		util.Debugf("fattrs = %#v", fattrs)
	}

	return fattrs, fh, nil
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
