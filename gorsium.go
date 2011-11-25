package main

import (
	"bufio"
	"os"
	"flag"
	"fmt"
	"path"
	"io/ioutil"
	"rsync"
	"fatal"
)

func main() {
	defer fatal.HandleFatal()

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
		if err != nil { fatal.Fail(err) }
		defer func() {
			errl := err
			if errl == nil && *backup { errl = os.Rename(basep, basep + "~") }
			if errl == nil {
				errl = os.Rename(tgtf.Name(), basep)
			} else {
				os.Remove(tgtf.Name())
			}
			if err == nil && errl != nil { fatal.Fail(errl) }
		}()
	case 3:
		basep = flag.Args()[1]
		tgtf, err = os.OpenFile(flag.Args()[2], os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil { fatal.Fail(err) }
	default:
		flag.Usage()
		os.Exit(2)
	}

	basef0, err := os.Open(basep)
	if err != nil { fatal.Fail(err) }
	var basef *bufio.Reader
	basef, err = bufio.NewReaderSize(basef0, *blocksize)
	if err != nil { fatal.Fail(err) }

	t := rsync.SumTableOf(basef, *blocksize)
	var srcf0 *os.File
	srcf0, err = os.Open(flag.Args()[0])
	if err != nil { fatal.Fail(err) }
	var d rsync.Delta
	d, err = t.Delta(bufio.NewReader(srcf0))

	if (*debug) {
		for _, e := range d {
			switch et := e.(type) {
			case rsync.Byterange:
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
		err = rsync.Patch(tgtf, basef0, &d)
	}

	if err != nil { fatal.Fail(err) }

	var fi *os.FileInfo
	fi, err = srcf0.Stat()
	if err != nil { fatal.Fail(err) }
	err = tgtf.Chmod(fi.Permission())
	if err != nil { fatal.Fail(err) }
}
