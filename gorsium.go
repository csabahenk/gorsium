package main

import (
	"io"
	"bufio"
	"os"
	"log"
	"crypto/md5"
	"flag"
	"fmt"
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
		if err != nil && err != os.EOF { log.Fatal(err) }
		if w.blen == 0 { break }
		ws := w.Wsum()
		m, ok := t.table[ws]
		if !ok {
			m = make(map[string] int)
			t.table[ws] = m
		}
		ms := md5.New()
		ms.Write(w.buf[:w.blen])
		s := string(ms.Sum())
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
	ms := md5.New()
	ms.Write(w.buf[w.idx:w.blen])
	ms.Write(w.buf[:w.idx])
	s, ok := m[string(ms.Sum())]
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


func main() {
	blocksize := flag.Int("blocksize", 4096, "blocksize used in rsync algorithm")
	debug := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	basef0, err := os.Open(flag.Args()[0])
	if err != nil { log.Fatal(err) }
	basef, err := bufio.NewReaderSize(basef0, *blocksize)
	if err != nil { log.Fatal(err) }

	t := SumTableOf(basef, *blocksize)
	in := bufio.NewReader(os.Stdin)
	d, err := t.Delta(in)

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
		err = Patch(os.Stdout, basef0, &d)
	}

	if err != nil { log.Fatal(err) }
}
