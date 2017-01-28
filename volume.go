package nfs

import "github.com/davecheney/nfs/rpc"

type Volume struct {
	*rpc.Client
	auth    rpc.Auth
	fh      []byte
	dirPath string
}

type MkdirArgs struct {
	rpc.Header
	Where Diropargs3
	Attrs Sattr3
}

func (v *Volume) Mkdir(name string) error {
	_, err := v.Call(&MkdirArgs{
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
			Filename: name,
		},
		Attrs: Sattr3{},
	})

	return err
}

func (v *Volume) DialNFS(nt, addr string) error {
	// get NFS port
	m := rpc.Mapping{
		Prog: NFS3_PROG,
		Vers: NFS3_VERS,
		Prot: rpc.IPPROTO_TCP,
		Port: 0,
	}

	var err error
	v.Client, err = DialService(nt, addr, m)
	if err != nil {
		return err
	}

	return nil
}
