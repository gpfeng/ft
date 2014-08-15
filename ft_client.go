// file transfer client 
// author: fengguangpu@nibirutech.com

package main

import (
    "fmt"
    "io"
    "os"
    "flag"
    "bufio"
    "path"
    "strings"
    "sync"
    "net"
    "log"
    "./util"
)

func ParseParams() (host string, port int, help, compressed bool, files []string) {
    H := flag.String("H", "127.0.0.1", "host name or ip address")
    p := flag.Int("p", 12345, "host port")
    c := flag.Bool("c", false, "compress or not")
    h := flag.Bool("h", false, "print help information")

    flag.Parse()

    host = *H
    port = *p
    compressed = *c
    help = *h
    files = flag.Args()

    return
}

// server will send only one message to client
func ReceiveResponse(conn net.Conn, filename string, end *bool, wg *sync.WaitGroup) {
    defer wg.Done()
    buf := make([]byte, util.MAX_RESPONSE_SIZE)
    l, err := conn.Read(buf)
    if err != nil || l <= 0 {
        log.Printf("Read() failed: %s\n", err.Error())
    } else {
        fields := strings.Split(string(buf), "/")
        if len(fields) < 2 {
            log.Printf("wrong response, must be a bug\n")
            *end = true
            return
        }
        if fields[0] == "ok" {
            log.Printf("Sending %s successed", filename)
        } else {
            log.Printf("Sending %s failed: %s", filename, fields[1])
            *end = true
        }
    }
}

// two phase
// 1. send file meta
// 2. send file contents
func Send(conn net.Conn, file *os.File, fullname string, compressed bool) {
    // get file information
    fi, err := file.Stat()
    if err != nil {
        log.Printf("Stat() %s failed: %s\n", file.Name(), err.Error())
        return
    }

    if fi.IsDir() {
        log.Printf("%s is a directory, not a file\n", fullname)
        return
    }

    size := fi.Size()
    mode := fi.Mode()

    md5, err := util.MD5(file)
    if err != nil {
       log.Printf(err.Error()) 
       return
    }

    meta := make([]byte, util.META_BUF_SIZE)
    if compressed {
        util.FormatMeta(meta, fullname, size, mode, md5, "yes", "zlib")
    } else {
        util.FormatMeta(meta, fullname, size, mode, md5, "no", "none")
    }

    // sent meta info
    if _, err := conn.Write([]byte(meta)); err != nil {
        log.Printf("Send meta failed: %s\n", err.Error())
        return
    }

    // stop if server has sent response 
    interrupted := false

    var wg sync.WaitGroup
    wg.Add(1)
    go ReceiveResponse(conn, fullname, &interrupted, &wg)

    // send file contents
    var sent int64 = 0
    buf := make([]byte, util.DATA_BUF_SIZE)
    hdr := make([]byte, util.DATA_HDR_SIZE)

    var fr *bufio.Reader

    if compressed {
        cr := util.CompressReader(file)
        fr = bufio.NewReader(cr)
        defer cr.Close()
    } else {
        fr = bufio.NewReader(file)
    }

    for {
        if interrupted { break }

        rlen, err := fr.Read(buf)
        if err == nil {
            if rlen > 0 {
                // send header
                util.FormatHeader(hdr, rlen)
                hlen, e := conn.Write(hdr)
                if e != nil || hlen != util.DATA_HDR_SIZE {
                    log.Printf("Write() failed: %s\n", e.Error())
                    break
                }
                // send data
                wlen, e := conn.Write(buf[:rlen])
                if e != nil || wlen != rlen {
                    log.Printf("Write() failed: %s\n", e.Error())
                    break
                }
                sent += int64(wlen)
            }
        } else if err == io.EOF {
            // send header, tell the server to stop
            util.FormatHeader(hdr, 0)
            hlen, e := conn.Write(hdr)
            if e != nil || hlen != util.DATA_HDR_SIZE {
                log.Printf("Write() failed: %s\n", e.Error())
                break
            }
            break
        } else {
            log.Printf("Read() failed: %s\n", err.Error())
            break
        }
    }

    if !interrupted {
        if compressed {
            log.Printf("%s: %d bytes, compressed to %d bytes, md5: %s\n",fullname, size, sent, md5)
        } else {
            log.Printf("%s: %d bytes, sent: %d bytes, md5: %s\n",fullname, size, sent, md5)
        }

    }
    wg.Wait()
}

func SendFile(name string, compressed bool, host string, port int, wg *sync.WaitGroup) {
    defer wg.Done()

    // open file
    file, err := os.Open(name)
    if err != nil {
        log.Printf("Open() %s failed: %s\n", name, err.Error())
        return
    }
    defer file.Close()

    // connect to the server
    conn, err := net.Dial("tcp",  fmt.Sprintf("%s:%d", host, port))
    if err != nil {
        log.Printf("Dial() failed: %s\n", err.Error())
        return
    }
    defer conn.Close()

    Send(conn, file, name, compressed)
}

func main() {
    var wg sync.WaitGroup

    host, port, help, compressed, files := ParseParams()

    if help {
        fmt.Printf("Usage: %s [-Hchp] file1 [file2 ...]\n", path.Base(os.Args[0]))
        flag.PrintDefaults()
        return
    }

    for _, file := range files {
        wg.Add(1)
        go SendFile(file, compressed, host, port, &wg)
    }

    wg.Wait()
}
