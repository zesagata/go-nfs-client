package main

import (
	"log"

	"github.com/davecheney/nfs"
	"github.com/davecheney/nfs/rpc"
)

func main() {
	mount, err := nfs.DialMount("tcp", "127.0.0.1")
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}
	defer mount.Close()

	auth := &rpc.AUTH_UNIX{
		Stamp:       0x017bbf7f,
		Machinename: "hasselhoff",
		Uid:         1001,
		Gid:         1001,
		GidLen:      1,
	}

	v, err := mount.Mount("/home/fahmed/f", auth.Auth())
	if err != nil {
		log.Fatalf("unable to mount volume: %v", err)
	}
	defer v.Close()

	if err = v.Mkdir("floob", 0775); err != nil {
		log.Fatalf("mkdir error: %v", err)
	}

	if err = mount.Unmount(); err != nil {
		log.Fatalf("unable to umount target: %v", err)
	}
	mount.Close()
}
