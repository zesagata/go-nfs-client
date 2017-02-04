package nfs

import (
	"bytes"
	"io"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/util"
	"github.com/davecheney/nfs/xdr"
)

type NFSFileReader struct {
	*Target

	// current position of the reader
	curr   uint32
	fsinfo *FSInfo

	// filehandle to the file
	fh string
}

func (rdr *NFSFileReader) Read(p []byte) (int, error) {
	util.Debugf("read called")
	type ReadArgs struct {
		rpc.Header
		FH     string
		Offset uint64
		Count  uint32
	}

	type ReadRes struct {
		Follows uint32
		Attrs   struct {
			Attrs Fattr
		}
		Count uint32
		Eof   uint32
		Data  struct {
			Length uint32
		}
	}

	buf, err := rdr.Call(&ReadArgs{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    NFS3_PROG,
			Vers:    NFS3_VERS,
			Proc:    NFSPROC3_READ,
			Cred:    rdr.auth,
			Verf:    rpc.AUTH_NULL,
		},
		FH:     rdr.fh,
		Offset: uint64(rdr.curr),
		Count:  rdr.fsinfo.RTPref,
	})

	if err != nil {
		util.Debugf("read(%x): %s", rdr.fh, err.Error())
		return 0, err
	}

	res, buf := xdr.Uint32(buf)
	if err = NFS3Error(res); err != nil {
		return 0, err
	}

	r := bytes.NewBuffer(buf)
	readres := &ReadRes{}
	if err = xdr.Read(r, readres); err != nil {
		return 0, err
	}

	util.Debugf("readres = %#v", readres)

	rdr.curr = rdr.curr + readres.Data.Length
	n, err := r.Read(p[:readres.Data.Length])
	if err != nil {
		return n, err
	}

	if readres.Eof != 0 {
		util.Debugf("eof = 0x%x", readres.Eof)
		err = io.EOF
	}

	return n, err
}

func (v *Target) Read(path string) (io.Reader, error) {
	_, fh, err := v.Lookup(path)
	if err != nil {
		return nil, err
	}

	rdr := &NFSFileReader{
		Target: v,
		fsinfo: v.fsinfo,
		fh:     string(fh),
	}
	util.Debugf("rdr = %#v", rdr)

	return rdr, nil
}
