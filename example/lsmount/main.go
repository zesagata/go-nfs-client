package main

import (
	"bytes"
	"crypto/sha256"
	"io"
	"io/ioutil"
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

	if _, err = v.Mkdir(dir, 0775); err != nil {
		log.Fatalf("mkdir error: %v", err)
	}

	if _, err = v.Mkdir(dir, 0775); err == nil {
		log.Fatalf("mkdir expected error")
	}

	// create a temp file
	f, err := os.Open("/dev/urandom")
	if err != nil {
		log.Fatalf("error openning random: %s", err.Error())
	}

	wr, err := v.Write("data", 0777)
	if err != nil {
		log.Fatalf("write fail: %s", err.Error())
	}

	// calculate the sha
	h := sha256.New()
	t := io.TeeReader(f, h)

	// Copy 20MB
	_, err = io.CopyN(wr, t, 20*1024*1024)
	if err != nil {
		log.Fatalf("error copying: %s", err.Error())
	}
	expectedSum := h.Sum(nil)

	//
	// get the file we wrote and calc the sum
	rdr, err := v.Read("data")
	if err != nil {
		log.Fatalf("read error: %v", err)
	}

	h = sha256.New()
	t = io.TeeReader(rdr, h)

	_, err = ioutil.ReadAll(t)
	if err != nil {
		log.Fatalf("readall error: %v", err)
	}
	actualSum := h.Sum(nil)

	if bytes.Compare(actualSum, expectedSum) != 0 {
		log.Fatalf("sums didn't match. actual=%x expected=%s", actualSum, expectedSum) //  Got=0%x expected=0%x", string(buf), testdata)
	}
	log.Printf("Sums match %x %x", actualSum, expectedSum)

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
