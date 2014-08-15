// file transfer server
// author: fengguangpu@nibirutech.com

package main

import (
	"./util"
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strconv"
)

func ParseParams() (port int, dst string, help bool) {
	p := flag.Int("p", 12345, "listen port")
	d := flag.String("d", ".", "target direcotry to store received file(s)")
	h := flag.Bool("h", false, "print help information")
	flag.Parse()

	return *p, *d, *h
}

func SendError(conn net.Conn, msg string, detail string) {
	log.Printf("%s: %s\n", msg, detail)
	_, err := conn.Write([]byte(fmt.Sprintf("error/server %s", msg)))
	if err != nil {
		log.Printf("send response failed\n")
		return
	}
}

func SendOk(conn net.Conn) {
	_, err := conn.Write([]byte("ok/finished"))
	if err != nil {
		log.Printf("send response failed\n")
		return
	}
}

func Checksum(file *os.File, fullname string, size int64, md5 string, conn net.Conn) {
	// check file size
	fi, err := file.Stat()
	if err != nil {
		SendError(conn, "Stat() failed", err.Error())
		return
	}
	if size != fi.Size() {
		log.Printf("Receiving %s failed\n", fullname)
		SendError(conn, "different size", fmt.Sprintf("sent: %d bytes, received: %d bytes", size, fi.Size()))
		return
	}

	// check file md5
	MD5, err := util.MD5(file)
	if err != nil {
		log.Printf(err.Error())
	}

	if md5 == MD5 && MD5 != "nil" {
		log.Printf("Receiving %s successed, md5(%s) matched", fullname, md5)
		SendOk(conn)
	} else if md5 != MD5 && md5 != "nil" && MD5 != "nil" {
		log.Printf("Receiving %s failed\n", fullname)
		SendError(conn, "different md5", fmt.Sprintf("sent: %s, received: %s", md5, MD5))
	} else {
		log.Printf("Warnig: please check the md5 of %s manually\n", fullname)
	}
}

func ReceiveData(conn net.Conn, hdr, buf []byte) (int, error) {
	// read header
	hl, err := conn.Read(hdr)
	if err != nil {
		return 0, err
	}

	if hl != util.DATA_HDR_SIZE {
		return 0, errors.New("wrong data header")
	}

	dl, err := util.ParseHeader(hdr)
	if err != nil || dl < 0 {
		return 0, errors.New("wrong data header")
	}

	start := 0
	left := dl
	for {
		if left == 0 {
			break
		}
		// read data
		b := buf[start:left]
		l, err := conn.Read(b)

		if err != nil {
			return 0, err
		}
		start += l
		left -= l
	}

	return dl, nil
}

func HandleClient(conn net.Conn, dst string) {
	addr := conn.RemoteAddr()
	log.Printf("%s connected to the server\n", addr.String())

	var meta util.FileMeta
	metabuf := make([]byte, util.META_BUF_SIZE)

	// read meta from network
	// meta fields:
	//  filename
	//  filesize
	//  filemode
	//  md5sum
	//  compressed
	//  algorithm

	metalen, err := conn.Read(metabuf)
	if err != nil && err != io.EOF {
		SendError(conn, "Meta Read() failed", err.Error())
		return
	}

	if metalen != util.META_BUF_SIZE {
		SendError(conn, "Read metadata failed", "client disconnect")
		return
	}

	err = util.ParseMeta(metabuf, &meta)
	if err != nil {
		SendError(conn, err.Error(), "must be a bug")
		return
	}

	// create target file
	filename := dst + "/" + meta.Name
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, meta.Mode)
	if err != nil {
		SendError(conn, "Create() failed", err.Error())
		return
	}
	defer file.Close()

	//log.Printf("received meta:\n%v", meta)

	var fw *bufio.Writer
	if meta.Compressed == "yes" {
		// decompress the received data stream and write to the target file
		dw := util.DecompressWriter(file)
		fw = bufio.NewWriter(dw)
		defer dw.Close()
	} else {
		fw = bufio.NewWriter(file)
	}

	// receive file contents
	var received int64 = 0
	buf := make([]byte, util.DATA_BUF_SIZE)
	hdr := make([]byte, util.DATA_HDR_SIZE)

	for {
		n, err := ReceiveData(conn, hdr, buf)
		if err != nil {
			SendError(conn, "ReceiveData() failed", err.Error())
			return
		}
		if n == 0 {
			break
		}
		received += int64(n)
		//fmt.Printf("[%d, %d]--", n, received)

		wl, err := fw.Write(buf[:n])
		if err != nil || wl != n {
			SendError(conn, "Write() failed", err.Error())
			return
		}
	}

	if err = fw.Flush(); err != nil {
		SendError(conn, "Flush() failed", err.Error())
		return
	}

	Checksum(file, filename, meta.Size, meta.Md5, conn)
}

func main() {
	port, dst, help := ParseParams()

	if help {
		fmt.Printf("Usage: %s [-dhp]\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		return
	}

	fi, err := os.Stat(dst)
	if err != nil {
		log.Printf("Stat() failed: %s\n", err.Error())
		return
	}
	if !fi.IsDir() {
		log.Printf("%s is not a directory\n", err.Error())
		return
	}

	ln, err := net.Listen("tcp", string(":"+strconv.Itoa(port)))
	if err != nil {
		log.Printf("Listen() failed: %s\n", err.Error())
		return
	}
	defer ln.Close()

	log.Printf("Server listening on port %d. received files will be put in [%s]\n", port, dst)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept() failed: %s\n", err.Error())
			continue
		}
		go HandleClient(conn, dst)
	}
}
