package nfs

import (
	"io"
	"os"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/util"
)

type FileWriter struct {
	*Target

	// current position of the writer
	curr   uint64
	fsinfo *FSInfo

	// filehandle to the file we're writing
	fh []byte
}

func (wr *FileWriter) Write(p []byte) (int, error) {
	type WriteArgs struct {
		rpc.Header
		FH     string
		Offset uint64
		Count  uint32

		// UNSTABLE(0), DATA_SYNC(1), FILE_SYNC(2) default
		How      uint32
		Contents []byte
	}

	var byteswritten int
	totalToWrite := len(p)

	// keep calling write in the case our buffer is larger than the page size
	for {

		if byteswritten == totalToWrite {
			break
		}

		writeSize := uint64(min(uint64(wr.fsinfo.WTPref), uint64(len(p))))
		segment := p[byteswritten:writeSize]
		util.Debugf("write(%x) len %d", wr.fh, writeSize)

		_, err := wr.call(&WriteArgs{
			Header: rpc.Header{
				Rpcvers: 2,
				Prog:    NFS3_PROG,
				Vers:    NFS3_VERS,
				Proc:    NFSPROC3_WRITE,
				Cred:    wr.auth,
				Verf:    rpc.AUTH_NULL,
			},
			FH:       string(wr.fh),
			Offset:   wr.curr,
			Count:    uint32(len(segment)),
			How:      2,
			Contents: segment,
		})

		if err != nil {
			util.Debugf("write(%x): %s", wr.fh, err.Error())
			return byteswritten, err
		}

		wr.curr = wr.curr + uint64(len(segment))
		byteswritten = byteswritten + len(segment)
	}

	return byteswritten, nil
}

func (wr *FileWriter) Close() error {
	type CommitArg struct {
		rpc.Header
		FH     []byte
		Offset uint64
		Count  uint32
	}

	_, err := wr.call(&CommitArg{
		Header: rpc.Header{
			Rpcvers: 2,
			Prog:    NFS3_PROG,
			Vers:    NFS3_VERS,
			Proc:    NFSPROC3_COMMIT,
			Cred:    wr.auth,
			Verf:    rpc.AUTH_NULL,
		},
		FH: wr.fh,
	})

	if err != nil {
		util.Debugf("commit(%x): %s", wr.fh, err.Error())
		return err
	}

	return nil
}

// Write writes to an existing file at path
func (v *Target) Write(path string, mode uint32) (io.WriteCloser, error) {
	_, fh, err := v.Lookup(path)
	if err != nil {
		if os.IsNotExist(err) {
			fh, err = v.Create(string(v.fh), path, mode)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	wr := &FileWriter{
		Target: v,
		fsinfo: v.fsinfo,
		fh:     fh,
	}

	return wr, nil
}
