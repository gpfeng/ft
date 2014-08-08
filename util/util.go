// file transfer utility 
// author: fengguangpu@nibirutech.com

package util

import (
    "fmt"
    "io"
    "os"
    "log"
    "path"
    "bufio"
    "errors"
    "strings"
    "strconv"
    "crypto/md5"
    "compress/zlib"
)

const (
    DATA_HDR_SIZE   = 8
    META_BUF_SIZE   = 512
    DATA_BUF_SIZE   = 10240
    DATA_RD_SIZE    = 1024

    MAX_RESPONSE_SIZE = 512
)

type FileMeta struct {
    Name string
    Size int64
    Mode os.FileMode
    Md5 string
    Compressed string   // "yes" or "no"
    Algorithm string    // compress algorithm
}

func FormatMeta(buf []byte, n string, s int64, mode os.FileMode, md5, com, alg string) {
    meta :=
      path.Base(n) + "\n" +
      strconv.FormatInt(s, 10) + "\n" +
      strconv.FormatUint(uint64(mode), 10) + "\n" +
      md5 + "\n" +
      com + "\n" +
      alg + "\n"

    for i, _ := range buf {
        buf[i] = 'x'
    }

    for i, v := range meta {
        buf[i] = byte(v)
    }
}

func ParseMeta(buf []byte, meta *FileMeta) error {

    var err error
    var m uint64

    // parse metadata
    fields := strings.Split(string(buf), "\n")
    if len(fields) < 6 {
        return errors.New("wrong metadata")
    }

    meta.Name = fields[0]

    meta.Size, err = strconv.ParseInt(fields[1], 10, 64)
    if err != nil {
        return errors.New("wrong metadata")
    }

    m, err = strconv.ParseUint(fields[2], 10, 64)
    if err != nil {
        return errors.New("wrong metadata")
    }
    meta.Mode = os.FileMode(m)

    meta.Md5 = fields[3]
    meta.Compressed = fields[4]
    meta.Algorithm = fields[5]

    return nil
}

func FormatHeader(hdr []byte, size int) {
    str := strconv.Itoa(size) + "\n"
    for i, _ := range hdr {
        hdr[i] = 'x'
    }
    for i, v := range str {
        hdr[i] = byte(v)
    }
}

func ParseHeader(buf []byte) (size int, err error) {
    fields := strings.Split(string(buf), "\n")
    if len(fields) < 1 {
        return -1, errors.New("wrong data header")
    }
    s, e := strconv.Atoi(fields[0])
    if e != nil {
        return -1, errors.New("wront data header")
    }
    return s, nil
}

// in file -> compress -> pw -> pr(out)
func CompressReader(in io.Reader) (out io.ReadCloser) {
    pr, pw := io.Pipe()

    go func() {
        zw := zlib.NewWriter(pw)
        bufin := bufio.NewReader(in)
        _, err := bufin.WriteTo(zw);
        if  err != nil {
            log.Fatal(err.Error())
        }
        zw.Close()
        pw.Close()
    } ()

    return pr
}

// pw(in) -> pr -> decompress -> out file
func DecompressWriter(out io.Writer) (in io.WriteCloser) {
    pr, pw := io.Pipe()

    go func() {
        zr, err := zlib.NewReader(pr)
        if err != nil {
            log.Fatal(err.Error())
        }
        bufout := bufio.NewWriter(out)
        _, err = bufout.ReadFrom(zr);
        if err != nil {
            log.Fatal(err.Error())
        }
        pr.Close()
        zr.Close()
        bufout.Flush()
    } ()

   return pw
}

// md5sum of a file
// warning: file pointer will point to the beginning of the file after 
//   calling this function
func MD5(in *os.File) (v string, err error)  {
    m := md5.New()
    // set it the the beginning of the file
    p, err := in.Seek(0, 0)
    if p != 0 || err != nil {
        return "nil", errors.New(fmt.Sprintf("Seek() failed in MD5: %s\n", err.Error()))
    }
    if _, err := io.Copy(m, in); err != nil {
        return "nil", errors.New(fmt.Sprintf("io.Copy() failed in MD5: %s\n", err.Error()))
    }
    // set it the the beginning of the file
    p, err = in.Seek(0, 0)
    if p != 0 || err != nil {
        return "nil", errors.New(fmt.Sprintf("Seek() failed in MD5: %s\n", err.Error()))
    }
    return fmt.Sprintf("%x", m.Sum(nil)), nil
}

func test() {
    fa, err := os.Open("a.txt")
    if err != nil {
        log.Fatal(err.Error())
    }
    defer fa.Close()

    fb, err := os.OpenFile("b.txt", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
    if err != nil {
        log.Fatal(err.Error())
    }

    in := CompressReader(fa)
    defer in.Close()

    //out := decompress(os.Stdout)
    out := DecompressWriter(fb)
    //out := fb

    //fmt.Printf("in: %v\nout: %v\n", in == nil, out == nil)

    var size int = 0
    buf := make([]byte, DATA_BUF_SIZE)
    for {
        rl, re := in.Read(buf)
        size += rl
        fmt.Printf("[%d %d] - ", rl, size)
        if re == nil || re == io.EOF {
            if rl > 0 {
                wl, we := out.Write(buf[:rl])
                if we != nil || rl != wl {
                    log.Printf(we.Error())
                    break
                }
            }
            if re == io.EOF {
                fmt.Printf("read finished\n")
                break
            }
        } else {
            log.Printf(re.Error())
            break
        }
    }
    out.Close()
}
