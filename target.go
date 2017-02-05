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

// wraps the Call function to check status and decode errors
func (v *Target) call(c interface{}) ([]byte, error) {
	buf, err := v.Call(c)
	if err != nil {
		return nil, err
	}

	res, buf := xdr.Uint32(buf)
	if err = NFS3Error(res); err != nil {
		return nil, err
	}

	return buf, nil
}

func (v *Target) FSInfo() (*FSInfo, error) {
	type FSInfoArgs struct {
		rpc.Header
		FsRoot []byte
	}

	buf, err := v.call(&FSInfoArgs{
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

	buf, err := v.call(&Lookup3Args{
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

	buf, err := v.call(&ReadDirPlus3Args{
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

	r := bytes.NewBuffer(buf)
	dirlist := &DirListOK{}
	if err = xdr.Read(r, dirlist); err != nil {
		return nil, err
	}

	return dirlist.DirListPlus.Entries, nil
}

// Creates a directory of the given name and returns its handle
func (v *Target) Mkdir(path string, perm os.FileMode) ([]byte, error) {
	type MkdirArgs struct {
		rpc.Header
		Where Diropargs3
		Attrs Sattr3
	}

	buf, err := v.call(&MkdirArgs{
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
		return nil, err
	}

	var fh []byte
	follows, buf := xdr.Uint32(buf)
	if follows != 0 {
		fh, buf = xdr.Opaque(buf)
	}

	util.Debugf("mkdir(%s): created successfully (0x%x)", path, fh)
	return fh, nil
}

func (v *Target) RmDir(path string) error {
	type RmDir3Args struct {
		rpc.Header
		Object Diropargs3
	}

	_, err := v.call(&RmDir3Args{
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

// create a file with name in the given directory specific by the fh with the given mode
func (v *Target) Create(fh string, name string, mode uint32) ([]byte, error) {
	type How struct {
		// 0 : UNCHECKED (default)
		// 1 : GUARDED
		// 2 : EXCLUSIVE
		Mode uint32
		Attr Sattr3
	}
	type Create3Args struct {
		rpc.Header
		Where Diropargs3
		HW    How
	}

	type Create3Res struct {
		Follows uint32
		FH      string
	}

	buf, err := v.call(&Create3Args{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    NFS3_PROG,
			Vers:    NFS3_VERS,
			Proc:    NFSPROC3_CREATE,
			Cred:    v.auth,
			Verf:    rpc.AUTH_NULL,
		},
		Where: Diropargs3{
			FH:       []byte(fh),
			Filename: name,
		},
		HW: How{
			Attr: Sattr3{
				Mode: SetMode{
					Set:  uint32(1),
					Mode: mode,
				},
			},
		},
	})

	if err != nil {
		util.Debugf("create(%s): %s", name, err.Error())
		return nil, err
	}

	res := &Create3Res{}
	r := bytes.NewBuffer(buf)
	if err = xdr.Read(r, res); err != nil {
		return nil, err
	}

	util.Debugf("create(%s): createted successfully", name)
	return []byte(res.FH), nil
}
