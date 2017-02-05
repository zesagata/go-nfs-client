package nfs

import (
	"bytes"
	"io"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/util"
	"github.com/davecheney/nfs/xdr"
)

type FileReader struct {
	*Target

	// current position of the reader
	curr   uint64
	fsinfo *FSInfo

	// filehandle to the file
	fh string
}

func (rdr *FileReader) Read(p []byte) (int, error) {
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

	readSize := uint32(min(uint64(rdr.fsinfo.RTPref), uint64(len(p))))
	util.Debugf("read(%x) len=%d offset=%d", rdr.fh, readSize, rdr.curr)

	buf, err := rdr.call(&ReadArgs{
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
		Count:  readSize,
	})

	if err != nil {
		util.Debugf("read(%x): %s", rdr.fh, err.Error())
		return 0, err
	}

	r := bytes.NewBuffer(buf)
	readres := &ReadRes{}
	if err = xdr.Read(r, readres); err != nil {
		return 0, err
	}

	rdr.curr = rdr.curr + uint64(readres.Data.Length)
	n, err := r.Read(p[:readres.Data.Length])
	if err != nil {
		return n, err
	}

	if readres.Eof != 0 {
		err = io.EOF
	}

	return n, err
}

func (v *Target) Read(path string) (io.Reader, error) {
	_, fh, err := v.Lookup(path)
	if err != nil {
		return nil, err
	}

	rdr := &FileReader{
		Target: v,
		fsinfo: v.fsinfo,
		fh:     string(fh),
	}

	return rdr, nil
}

func min(x, y uint64) uint64 {
	if x > y {
		return y
	}
	return x
}
