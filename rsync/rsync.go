/* rsync routines based on
 * http://blog.liw.fi/posts/rsync-in-python/ and
 * http://rsync.samba.org/tech_report/node3.html
 */
package rsync

import (
	"bufio"
	"io"

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
	if blen <= 0 {
		blen = len(buf)
	}
	return &WsumBuf{buf: buf, blen: blen}
}

func (w *WsumBuf) Fill(r io.Reader) (err error) {
	w.blen, err = r.Read(w.buf)

	if err != nil {
		if err == io.EOF {
			w.blen = 0
		}
		return
	}
	if w.blen != len(w.buf) {
		err = io.EOF
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
		w.whi += uint16((len(w.buf) - i)) * uint16(c)
	}
}

func (w *WsumBuf) Wsum() uint32 {
	return (uint32(w.whi) << 16) | uint32(w.wlow)
}

func (w *WsumBuf) Update(c byte) byte {
	w.wlow = w.wlow - uint16(w.buf[w.idx]) + uint16(c)
	w.whi = w.whi - uint16(len(w.buf))*uint16(w.buf[w.idx]) + w.wlow
	oldc := w.buf[w.idx]
	w.buf[w.idx] = c
	w.idx = (w.idx + 1) % len(w.buf)
	return oldc
}

// SumTable

type SumTable struct {
	Blen  int
	Table map[uint32]map[string]int
}

func NewSumTable(blen int) *SumTable {
	t := SumTable{Blen: blen}
	t.Table = make(map[uint32]map[string]int)
	return &t
}

func SumTableOf(r io.Reader, l int) (*SumTable, error) {
	t := NewSumTable(l)
	if r == nil {
		return t, nil
	}

	w := NewWsumBuf(make([]byte, l), 0)
	for i := 0; ; i++ {
		err := w.Fill(r)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if w.blen == 0 {
			break
		}
		ws := w.Wsum()
		m, ok := t.Table[ws]
		if !ok {
			m = make(map[string]int)
			t.Table[ws] = m
		}
		s := string(md5.Md5(w.buf[:w.blen]))
		if _, ok := m[s]; !ok {
			m[s] = i
		}
	}

	return t, nil
}

func (t *SumTable) Get(buf []byte, blen int) (int, bool) {
	w := NewWsumBuf(buf, blen)
	w.CalcWsum()
	return t.GetWsumBuf(w)
}

func (t *SumTable) GetWsumBuf(w *WsumBuf) (int, bool) {
	m, ok := t.Table[w.Wsum()]
	if !ok {
		return 0, ok
	}
	s, ok := m[string(md5.Md5_2(w.buf[w.idx:w.blen], w.buf[:w.idx]))]
	return s, ok
}

// Delta + Patch

type Byterange struct {
	Offset int64
	Blen   int
}

type Delta []interface{}

func (t *SumTable) Delta(f *bufio.Reader) (d Delta, err error) {
	w := NewWsumBuf(make([]byte, t.Blen), 0)

	b := make([]byte, 0, t.Blen)
	var i int
	var ok bool

readloop:
	for {
		if len(b) != 0 {
			b = make([]byte, 0, t.Blen)
		}

		err = w.Fill(f)
		if err != nil {
			if err == io.EOF && w.blen != 0 {
				i, ok = t.GetWsumBuf(w)
			}
			break
		}

		for {
			i, ok = t.GetWsumBuf(w)
			if ok {
				break
			}
			var c byte
			c, err = f.ReadByte()
			if err != nil {
				break readloop
			}
			b = append(b, w.Update(c))
		}

		if len(b) != 0 {
			d = append(d, b)
		}
		d = append(d, Byterange{int64(i) * int64(t.Blen), t.Blen})
	}

	if err != io.EOF {
		return
	}

	if len(b) != 0 {
		d = append(d, b)
	}
	if w.blen != 0 {
		if ok {
			d = append(d, Byterange{int64(i) * int64(t.Blen), w.blen})
		} else {
			b = append(w.buf[w.idx:w.blen], w.buf[:w.idx]...)
			d = append(d, b)
		}
	}

	return d, nil
}

func Patch(outf io.Writer, basef io.ReaderAt, d *Delta) (err error) {
	var b []byte
	for _, e := range *d {
		switch et := e.(type) {
		case Byterange:
			if cap(b) < et.Blen {
				b = make([]byte, et.Blen)
			}
			_, err = basef.ReadAt(b[:et.Blen], et.Offset)
			if err == nil {
				_, err = outf.Write(b[:et.Blen])
			}
		case []byte:
			_, err = outf.Write(et)
		default:
			panic("got lost")
		}
		if err != nil {
			return
		}
	}

	return
}
