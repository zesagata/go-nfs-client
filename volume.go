package nfs

import "github.com/davecheney/nfs/rpc"

type Volume struct {
	*rpc.Client

	auth    rpc.Auth
	fh      []byte
	dirPath string
}

func NewTarget(nt, addr string, auth rpc.Auth, fh []byte, dirpath string) (*Volume, error) {
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
	}

	return vol, nil
}
