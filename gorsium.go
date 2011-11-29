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
	"io"
)

func main() {
	defer fatal.HandleFatal()
	done := false

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] <src> <tgt-base> [<tgt>]\nOptions can be:\n", os.Args[0])
		flag.PrintDefaults()
	}
	blocksize := flag.Int("blocksize", 4096, "blocksize used in rsync algorithm")
	backup := flag.Bool("backup", false, "backup original of target file")
	debug := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	var basep string
	var basef io.Reader
	var tgtf *os.File
	var err os.Error
	switch len(flag.Args()) {
	case 2:
		basep = flag.Args()[1]
		tdir, tname := path.Split(basep)
		tgtf, err = ioutil.TempFile(tdir, tname + ".")
		if err != nil { fatal.Fail(err) }
		defer func() {
			var err os.Error
			if done { // sync terminated successfully, doing the final renames
				if *backup && basef != nil { err = os.Rename(basep, basep + "~") }
				if err == nil { err = os.Rename(tgtf.Name(), basep) }
			}
			if !done || err != nil { // either sync failed or renames failed, just do an emergency cleanup
				os.Remove(tgtf.Name())
			}
			if done && err != nil { // sync terminated but renames failed, raise the error
				fatal.Fail(err)
			}
			// either both sync and renames terminated successfully, then nothing to do,
			// or sync has failed, then we called from panic context, so that's taken care of,
			// nothing to do.
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
	if err == nil {
		basef, err = bufio.NewReaderSize(basef0, *blocksize)
	} else if perr, ok := err.(*os.PathError); ok && perr.Error == os.ENOENT {
		err = nil
	}
	if err != nil { fatal.Fail(err) }

	t := rsync.SumTableOf(basef, *blocksize)
	srcf0, err := os.Open(flag.Args()[0])
	if err != nil { fatal.Fail(err) }
	d, err := t.Delta(bufio.NewReader(srcf0))

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

	fi, err := srcf0.Stat()
	if err != nil { fatal.Fail(err) }
	err = tgtf.Chmod(fi.Permission())
	if err != nil { fatal.Fail(err) }

	done = true
}
