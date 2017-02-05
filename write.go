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

	totalToWrite := len(p)
	writeSize := uint64(min(uint64(wr.fsinfo.WTPref), uint64(totalToWrite)))

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
		Count:    uint32(writeSize),
		How:      2,
		Contents: p[:writeSize],
	})

	if err != nil {
		util.Debugf("write(%x): %s", wr.fh, err.Error())
		return int(writeSize), err
	}

	util.Debugf("write(%x) len=%d offset=%d written=%d total=%d",
		wr.fh, writeSize, wr.curr, writeSize, totalToWrite)

	wr.curr = wr.curr + writeSize

	return int(writeSize), nil
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
func (v *Target) Write(path string, perm os.FileMode) (io.WriteCloser, error) {
	_, fh, err := v.Lookup(path)
	if err != nil {
		if os.IsNotExist(err) {
			fh, err = v.Create(string(v.fh), path, perm)
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
