package nfs

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fdawg4l/nfs/rpc"
	"github.com/fdawg4l/nfs/util"
	"github.com/fdawg4l/nfs/xdr"
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

// Lookup returns attributes and the file handle to a given dirent
func (v *Target) Lookup(p string) (*Fattr, []byte, error) {
	var (
		err   error
		fattr *Fattr
		fh    = v.fh
	)

	// desecend down a path heirarchy to get the last elem's fh
	dirents := strings.Split(path.Clean(p), "/")
	for _, dirent := range dirents {
		// we're assuming the root is always the root of the mount
		if dirent == "." || dirent == "" {
			util.Debugf("root -> 0x%x", fh)
			continue
		}

		fattr, fh, err = v.lookup(fh, dirent)
		if err != nil {
			return nil, nil, err
		}

		//	util.Debugf("%s -> 0x%x", dirent, fh)
	}

	return fattr, fh, nil
}

// lookup returns the same as above, but by fh and name
func (v *Target) lookup(fh []byte, name string) (*Fattr, []byte, error) {
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
			FH:       fh,
			Filename: name,
		},
	})

	if err != nil {
		util.Debugf("lookup(%s): %s", name, err.Error())
		return nil, nil, err
	}

	fh, buf = xdr.Opaque(buf)
	util.Debugf("lookup(%s): FH 0x%x", name, fh)

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

func (v *Target) ReadDirPlus(dir string) ([]*EntryPlus, error) {
	_, fh, err := v.Lookup(dir)
	if err != nil {
		return nil, err
	}

	return v.readDirPlus(fh)
}

func (v *Target) readDirPlus(fh []byte) ([]*EntryPlus, error) {
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
		CookieVerf uint64
		Follows    uint32
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
		FH:       fh,
		DirCount: 512,
		MaxCount: 4096,
	})

	if err != nil {
		util.Debugf("readdir(%x): %s", fh, err.Error())
		return nil, err
	}

	// The dir list entries are so-called "optional-data".  We need to check
	// the Follows fields before continuing down the array.  Effectively, it's
	// an encoding used to flatten a linked list into an array where the
	// Follows field is set when the next idx has data. See
	// https://tools.ietf.org/html/rfc4506.html#section-4.19 for details.
	r := bytes.NewBuffer(buf)
	dirlistOK := &DirListOK{}
	if err = xdr.Read(r, dirlistOK); err != nil {
		return nil, err
	}

	if dirlistOK.Follows == 0 {
		return nil, nil
	}

	var entries []*EntryPlus
	for {
		var entry EntryPlus

		if err = xdr.Read(r, &entry); err != nil {
			return nil, err
		}

		entries = append(entries, &entry)

		if entry.ValueFollows == 0 {
			break
		}
	}

	// The last byte is EOF
	var eof uint32
	if err = xdr.Read(r, &eof); err != nil {
		util.Debugf("ReadDirPlus(0x%x): expected EOF", fh)
	}

	return entries, nil
}

// Creates a directory of the given name and returns its handle
func (v *Target) Mkdir(path string, perm os.FileMode) ([]byte, error) {
	dir, newDir := filepath.Split(path)
	_, fh, err := v.Lookup(dir)
	if err != nil {
		return nil, err
	}

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
			FH:       fh,
			Filename: newDir,
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

	follows, buf := xdr.Uint32(buf)
	if follows != 0 {
		fh, buf = xdr.Opaque(buf)
	}

	util.Debugf("mkdir(%s): created successfully (0x%x)", path, fh)
	return fh, nil
}

// Create a file with name the given mode
func (v *Target) Create(path string, perm os.FileMode) ([]byte, error) {
	dir, newFile := filepath.Split(path)
	_, fh, err := v.Lookup(dir)
	if err != nil {
		return nil, err
	}

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
		FH      []byte
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
			FH:       fh,
			Filename: newFile,
		},
		HW: How{
			Attr: Sattr3{
				Mode: SetMode{
					Set:  uint32(1),
					Mode: uint32(perm.Perm()),
				},
			},
		},
	})

	if err != nil {
		util.Debugf("create(%s): %s", path, err.Error())
		return nil, err
	}

	res := &Create3Res{}
	r := bytes.NewBuffer(buf)
	if err = xdr.Read(r, res); err != nil {
		return nil, err
	}

	util.Debugf("create(%s): created successfully", path)
	return res.FH, nil
}

// Remove a file
func (v *Target) Remove(path string) error {
	parentDir, deleteFile := filepath.Split(path)
	_, fh, err := v.Lookup(parentDir)
	if err != nil {
		return err
	}

	return v.remove(fh, deleteFile)
}

// remove the named file from the parent (fh)
func (v *Target) remove(fh []byte, deleteFile string) error {
	type RemoveArgs struct {
		rpc.Header
		Object Diropargs3
	}

	_, err := v.call(&RemoveArgs{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    NFS3_PROG,
			Vers:    NFS3_VERS,
			Proc:    NFSPROC3_REMOVE,
			Cred:    v.auth,
			Verf:    rpc.AUTH_NULL,
		},
		Object: Diropargs3{
			FH:       fh,
			Filename: deleteFile,
		},
	})

	if err != nil {
		util.Debugf("remove(%s): %s", deleteFile, err.Error())
		return err
	}

	return nil
}

// RmDir removes a non-empty directory
func (v *Target) RmDir(path string) error {
	dir, deletedir := filepath.Split(path)
	_, fh, err := v.Lookup(dir)
	if err != nil {
		return err
	}

	return v.rmDir(fh, deletedir)
}

// delete the named directory from the parent directory (fh)
func (v *Target) rmDir(fh []byte, name string) error {
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
			FH:       fh,
			Filename: name,
		},
	})

	if err != nil {
		util.Debugf("rmdir(%s): %s", name, err.Error())
		return err
	}

	util.Debugf("rmdir(%s): deleted successfully", name)
	return nil
}

func (v *Target) RemoveAll(path string) error {
	parentDir, deleteDir := filepath.Split(path)
	_, parentDirfh, err := v.Lookup(parentDir)
	if err != nil {
		return err
	}

	// Easy path.  This is a directory and it's empty.  If not a dir or not an
	// empty dir, this will throw an error.
	err = v.rmDir(parentDirfh, deleteDir)
	if err == nil || os.IsNotExist(err) {
		return nil
	}

	if IsNotDirError(err) {
		return err
	}

	return v.removeAll(parentDirfh, deleteDir)
}

// removeAll removes the deleteDir from the parentDir recursively
func (v *Target) removeAll(parentDirfh []byte, deleteDir string) error {

	_, deleteDirfh, err := v.lookup(parentDirfh, deleteDir)
	if err != nil {
		return err
	}

	// BFS the dir tree.  If dir, recurse, then delete the dir and all files.

	// This is a directory, get all of its Entries
	entries, err := v.readDirPlus(deleteDirfh)
	if err != nil {
		return err
	}

	for _, entry := range entries {

		// skip "." and ".."
		if entry.FileName == "." || entry.FileName == ".." {
			continue
		}

		if entry.Attr.Attr.Type == NF3DIR {
			if err = v.removeAll(deleteDirfh, entry.FileName); err != nil {
				return err
			}

			err = v.rmDir(deleteDirfh, entry.FileName)
		} else {
			err = v.remove(deleteDirfh, entry.FileName)
		}

		if err != nil {
			util.Errorf("%s %s", entry.FileName, err.Error())
			return err
		}
	}

	return nil
}
