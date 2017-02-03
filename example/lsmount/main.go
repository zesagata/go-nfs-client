package main

import (
	"log"
	"os"
	"strings"

	"github.com/davecheney/nfs"
	"github.com/davecheney/nfs/rpc"
)

func main() {

	host := strings.Split(os.Args[1], ":")[0]
	target := strings.Split(os.Args[1], ":")[1]

	mount, err := nfs.DialMount("tcp", host)
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

	v, err := mount.Mount(target, auth.Auth())
	if err != nil {
		log.Fatalf("unable to mount volume: %v", err)
	}
	defer v.Close()

	dir := os.Args[2]
	if err = v.Mkdir(dir, 0775); err != nil {
		log.Fatalf("mkdir error: %v", err)
	}

	if err = v.Mkdir(dir, 0775); err == nil {
		log.Fatalf("mkdir expected error")
	}

	if err = v.RmDir(dir); err != nil {
		log.Fatalf("rmdir error: %v", err)
	}

	if err = mount.Unmount(); err != nil {
		log.Fatalf("unable to umount target: %v", err)
	}
	mount.Close()
}
