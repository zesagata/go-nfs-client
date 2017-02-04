package nfs

import (
	"bytes"
	"os"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/util"
	"github.com/davecheney/nfs/xdr"
)

type Target struct {
	*rpc.Client

	auth    rpc.Auth
	fh      []byte
	dirPath string
	fsinfo  *FSInfo
}

func NewTarget(nt, addr string, auth rpc.Auth, fh []byte, dirpath string) (*Target, error) {
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

	vol := &Target{
		Client:  client,
		auth:    auth,
		fh:      fh,
		dirPath: dirpath,
	}

	fsinfo, err := vol.FSInfo()
	if err != nil {
		return nil, err
	}

	vol.fsinfo = fsinfo
	util.Debugf("%s:%s fsinfo=%#v", addr, dirpath, fsinfo)

	return vol, nil
}

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

func (v *Target) FSInfo() (*FSInfo, error) {
	type FSInfoArgs struct {
		rpc.Header
		FsRoot []byte
	}

	buf, err := v.Call(&FSInfoArgs{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    NFS3_PROG,
			Vers:    NFS3_VERS,
			Proc:    NFSPROC3_FSINFO,
			Cred:    v.auth,
			Verf:    rpc.AUTH_NULL,
		},
		FsRoot: v.fh,
	})

	if err != nil {
		util.Debugf("fsroot: %s", err.Error())
		return nil, err
	}

	res, buf := xdr.Uint32(buf)
	if err = NFS3Error(res); err != nil {
		return nil, err
	}

	fsinfo := new(FSInfo)
	r := bytes.NewBuffer(buf)
	if err = xdr.Read(r, fsinfo); err != nil {
		return nil, err
	}

	return fsinfo, nil
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
	}

	return fattrs, fh, nil
}

func (v *Target) ReadDirPlus(dir string) ([]EntryPlus, error) {
	_, fh, err := v.Lookup(dir)
	if err != nil {
		return nil, err
	}

	type ReadDirPlus3Args struct {
		rpc.Header
		FH         []byte
		Cookie     uint64
		CookieVerf uint64
		DirCount   uint32
		MaxCount   uint32
	}

	type DirListOK struct {
		DirAttrs struct {
			Follows  uint32
			DirAttrs Fattr
		}
		CookieVerf  uint64
		Follows     uint32
		DirListPlus struct {
			Entries []EntryPlus
			EOF     uint32
		}
	}

	buf, err := v.Call(&ReadDirPlus3Args{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    NFS3_PROG,
			Vers:    NFS3_VERS,
			Proc:    NFSPROC3_READDIRPLUS,
			Cred:    v.auth,
			Verf:    rpc.AUTH_NULL,
		},
		FH:       v.fh,
		DirCount: 512,
		MaxCount: 4096,
	})

	if err != nil {
		util.Debugf("readdir(%x): %s", fh, err.Error())
		return nil, err
	}

	res, buf := xdr.Uint32(buf)
	if err = NFS3Error(res); err != nil {
		return nil, err
	}

	r := bytes.NewBuffer(buf)
	dirlist := &DirListOK{}
	if err = xdr.Read(r, dirlist); err != nil {
		return nil, err
	}

	return dirlist.DirListPlus.Entries, nil
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
