package main

import (
	"fmt"
	"log"

	"github.com/davecheney/nfs"
	"github.com/davecheney/nfs/rpc"
)

func main() {
	pm, err := rpc.DialPortmapper("tcp", "127.0.0.1")
	if err != nil {
		log.Fatalf("unable to contact portmapper: %v", err)
	}
	// get MOUNT port
	m := rpc.Mapping{
		Prog: nfs.MOUNT_PROG,
		Vers: nfs.MOUNT_VERS,
		Prot: rpc.IPPROTO_TCP,
		Port: 0,
	}
	port, err := pm.Getport(m)
	if err != nil {
		log.Fatalf("unable to get MOUNT port: %v", err)
	}
	log.Println("MOUNT", port)
	defer pm.Close()
	mount, err := nfs.DialMount("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		log.Fatal("unable to dial MOUNT service: %v", err)
	}
	defer mount.Close()

	auth := &rpc.AUTH_UNIX{
		Stamp:       0x017bbf7f,
		Machinename: "hasselhoff",
		Uid:         0,
		Gid:         0,
		GidLen:      1,
	}

	v, err := mount.Mount("/home/fahmed/f", auth.Auth())
	if err != nil {
		log.Fatalf("unable to mount volume: %v", err)
	}

	if err = v.Unmount(); err != nil {
		log.Fatalf("unable to umount target: %v", err)
	}
}
