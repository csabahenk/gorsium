package main

import (
	"io"
	"bufio"
	"os"
	"flag"
	"fmt"
	"path"
	"io/ioutil"
	"md5"
)

// WsumBuf

type WsumBuf struct {
	buf       []byte
	blen      int
	idx       int
	whi, wlow uint16
}

func NewWsumBuf(buf []byte, blen int) *WsumBuf {
	if blen <= 0 { blen = len(buf) }
	return &WsumBuf{buf: buf, blen: blen}
}

func (w *WsumBuf) Fill(r io.Reader) (err os.Error) {
	w.blen, err = r.Read(w.buf)

	if err != nil {
		if err == os.EOF { w.blen = 0 }
		return
	}
	if w.blen != len(w.buf) {
		err = os.EOF
	}

	w.CalcWsum()

	return
}

func (w *WsumBuf) CalcWsum() {
	w.idx = 0
	w.wlow = 0
	w.whi = 0
	for i, c := range w.buf[:w.blen] {
		w.wlow += uint16(c)
		w.whi  += uint16((len(w.buf) - i)) * uint16(c)
	}
}

func (w *WsumBuf) Wsum() uint32 {
	return (uint32(w.whi) << 16) | uint32(w.wlow)
}

func (w *WsumBuf) Update(c byte) byte {
	w.wlow = w.wlow - uint16(w.buf[w.idx]) + uint16(c)
	w.whi = w.whi - uint16(len(w.buf)) * uint16(w.buf[w.idx]) + w.wlow
	oldc := w.buf[w.idx]
	w.buf[w.idx] = c
	w.idx = (w.idx + 1) % len(w.buf)
	return oldc
}

// SumTable

type SumTable struct {
	blen  int
	table map[uint32] map[string] int
}

func NewSumTable(blen int) *SumTable {
	t := SumTable{blen: blen}
	t.table = make(map[uint32] map[string] int)
	return &t
}

func SumTableOf(r io.Reader, l int) *SumTable {
	t := NewSumTable(l)

	w := NewWsumBuf(make([]byte, l), 0)
	for i := 0 ;; i++ {
		err := w.Fill(r)
		if err != nil && err != os.EOF { fail(err) }
		if w.blen == 0 { break }
		ws := w.Wsum()
		m, ok := t.table[ws]
		if !ok {
			m = make(map[string] int)
			t.table[ws] = m
		}
		s := string(md5.Md5(w.buf[:w.blen]))
		if _, ok := m[s]; !ok {
			m[s] = i
		}
	}

	return t
}

func (t *SumTable) Get(buf []byte, blen int) (int, bool) {
	w := NewWsumBuf(buf, blen)
	w.CalcWsum()
	return t.GetWsumBuf(w)
}

func (t *SumTable) GetWsumBuf(w *WsumBuf) (int, bool) {
	m, ok := t.table[w.Wsum()]
	if !ok { return 0, ok }
	s, ok := m[string(md5.Md5_2(w.buf[w.idx:w.blen], w.buf[:w.idx]))]
	return s, ok
}

// Delta + Patch

type byterange struct {
	offset int64
	blen int
}

type delta []interface{}

func (t *SumTable) Delta(f *bufio.Reader) (d delta, err os.Error) {
	w := NewWsumBuf(make([]byte, t.blen), 0)

	b := make([]byte, 0, t.blen)
	var i int
	var ok bool

 readloop:
	for {
		if len(b) != 0 {
			 b = make([]byte, 0, t.blen)
		}

		err = w.Fill(f)
		if err != nil {
			if  err == os.EOF && w.blen != 0 {
				i, ok = t.GetWsumBuf(w)
			}
			break
		}

		for  {
			i, ok = t.GetWsumBuf(w)
			if ok { break }
			var c byte
			c, err = f.ReadByte()
			if err != nil { break readloop }
			b = append(b, w.Update(c))
		}

		if len(b) != 0 {
			d = append(d, b)
		}
		d = append(d, byterange{ int64(i) * int64(t.blen), t.blen })
	}

	if err != os.EOF { return }

	if len(b) != 0 {
		d = append(d, b)
	}
	if w.blen != 0 {
		if ok {
			d = append(d, byterange{ int64(i) * int64(t.blen), w.blen })
		} else {
			b = append(w.buf[w.idx:w.blen], w.buf[:w.idx]...)
			d = append(d, b)
		}
	}

	return d, nil
}

func Patch(outf io.Writer, basef io.ReaderAt, d *delta) (err os.Error) {
	var b []byte
	for _, e := range *d {
		switch et := e.(type) {
		case byterange:
			if cap(b) < et.blen { b = make([]byte, et.blen) }
			_, err = basef.ReadAt(b[:et.blen], et.offset)
			if err == nil {
				_, err = outf.Write(b[:et.blen])
			}
		case []byte:
			_, err = outf.Write(et)
		default:
			panic("got lost")
		}
		if err != nil { return }
	}

	return
}


type fatal struct {
	err interface{}
}

func fail(err interface{}) {
	panic(fatal{err})
}

func handleFatal() {
	if err := recover(); err != nil {
		if f, ok := err.(fatal); ok {
			fmt.Println(f.err)
			os.Exit(1)
		}
		panic(err)
	}
}

func main() {
	defer handleFatal()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] <src> <tgt-base> [<tgt>]\nOptions can be:\n", os.Args[0])
		flag.PrintDefaults()
	}
	blocksize := flag.Int("blocksize", 4096, "blocksize used in rsync algorithm")
	backup := flag.Bool("backup", false, "backup original of target file")
	debug := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	var basep string
	var tgtf *os.File
	var err os.Error
	switch len(flag.Args()) {
	case 2:
		basep = flag.Args()[1]
		tdir, tname := path.Split(basep)
		tgtf, err = ioutil.TempFile(tdir, tname + ".")
		if err != nil { fail(err) }
		defer func() {
			errl := err
			if errl == nil && *backup { errl = os.Rename(basep, basep + "~") }
			if errl == nil {
				errl = os.Rename(tgtf.Name(), basep)
			} else {
				os.Remove(tgtf.Name())
			}
			if err == nil && errl != nil { fail(errl) }
		}()
	case 3:
		basep = flag.Args()[1]
		tgtf, err = os.OpenFile(flag.Args()[2], os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil { fail(err) }
	default:
		flag.Usage()
		os.Exit(2)
	}

	basef0, err := os.Open(basep)
	if err != nil { fail(err) }
	var basef *bufio.Reader
	basef, err = bufio.NewReaderSize(basef0, *blocksize)
	if err != nil { fail(err) }

	t := SumTableOf(basef, *blocksize)
	var srcf0 *os.File
	srcf0, err = os.Open(flag.Args()[0])
	if err != nil { fail(err) }
	var d delta
	d, err = t.Delta(bufio.NewReader(srcf0))

	if (*debug) {
		for _, e := range d {
			switch et := e.(type) {
			case byterange:
				fmt.Fprint(os.Stderr, et, " ")
			case []byte:
				fmt.Fprintf(os.Stderr, "%q ", string(et))
			}
		}
		//not nice enough
		//fmt.Fprintln(os.Stderr, t)
		fmt.Fprint(os.Stderr, "\n\n")
	}

	if err == nil {
		err = Patch(tgtf, basef0, &d)
	}

	if err != nil { fail(err) }

	var fi *os.FileInfo
	fi, err = srcf0.Stat()
	if err != nil { fail(err) }
	err = tgtf.Chmod(fi.Permission())
	if err != nil { fail(err) }
}
