package main

import (
	"io"
	"bufio"
	"os"
	"log"
	"fmt"
)

type WsumBuf struct {
	buf       []byte
	idx       int
	whi, wlow uint16
}

func NewWsumBuf(buf []byte) *WsumBuf {
	return &WsumBuf{buf: buf}
}

func (w *WsumBuf) Len() int {
	return len(w.buf)
}

func (w *WsumBuf) Fill(r io.Reader) (n int, err os.Error) {
	n, err = r.Read(w.buf)

	if err == nil && n != len(w.buf) {
		err = os.EOF
	}
	if err != nil {
		return
	}

	for i, c := range w.buf {
		w.wlow += uint16(c)
		w.whi  += uint16((len(w.buf) - i)) * uint16(c)
	}

	return
}

func (w *WsumBuf) Wsum() uint32 {
	return (uint32(w.whi) << 16) | uint32(w.wlow)
}

func (w *WsumBuf) Update(c byte) {
	w.wlow = w.wlow - uint16(w.buf[w.idx]) + uint16(c)
	w.whi = w.whi - uint16(len(w.buf)) * uint16(w.buf[w.idx]) + w.wlow
	w.buf[w.idx] = c
	w.idx = (w.idx + 1) % len(w.buf)
}

func main() {
	w := NewWsumBuf(make([]byte, 4096))

	in := bufio.NewReader(os.Stdin)

	_, err := w.Fill(in)
	if err != nil { log.Fatal(err) }

	for {
		c, err := in.ReadByte()
		if err != nil {
			if err == os.EOF { break }
			log.Fatal(err)
		}

		w.Update(c)
	}

	fmt.Println(w.Wsum())
}
