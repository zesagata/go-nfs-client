package main

import (
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/davecheney/nfs"
	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/util"
)

func main() {
	util.DefaultLogger.SetDebug(true)

	b := strings.Split(os.Args[1], ":")

	host := b[0]
	target := b[1]
	dir := os.Args[2]

	util.Infof("host=%s target=%s dir=%s\n", host, target, dir)

	mount, err := nfs.DialMount("tcp", host)
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}
	defer mount.Close()

	auth := &rpc.AUTH_UNIX{
		Stamp:       rand.New(rand.NewSource(time.Now().UnixNano())).Uint32(),
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

	if err = v.Mkdir(dir, 0775); err != nil {
		log.Fatalf("mkdir error: %v", err)
	}

	if err = v.Mkdir(dir, 0775); err == nil {
		log.Fatalf("mkdir expected error")
	}

	_, _, err = v.Lookup(dir)
	if err != nil {
		log.Fatalf("lookup error: %s", err.Error())
	}

	dirs, err := v.ReadDirPlus(dir)
	if err != nil {
		log.Fatalf("readdir error: %s", err.Error())
	}

	util.Infof("dirs:")
	for _, dir := range dirs {
		util.Infof("\t%s\t%d:%d\t0%o", dir.FileName, dir.Attr.Attr.UID, dir.Attr.Attr.GID, dir.Attr.Attr.Mode)
	}

	if err = v.RmDir(dir); err != nil {
		log.Fatalf("rmdir error: %v", err)
	}

	if err = mount.Unmount(); err != nil {
		log.Fatalf("unable to umount target: %v", err)
	}
	mount.Close()
}
