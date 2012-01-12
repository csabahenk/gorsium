package main

import (
	"bufio"
	"os"
	"flag"
	"fmt"
	"log"
	"strings"
	"exec"
	"rpc"
	"gob"
	"sync"
	"rsync"
)

var debug, zerosep *bool

func syncFile(cli *rpc.Client, file string) {
	var t *rsync.SumTable

	err := cli.Call("Server.Sumtable", file, &t)
	defer func() {
		if *zerosep {
			errs := ""
			if err != nil { errs = fmt.Sprint(err) }
			os.Stdout.Write([]byte(file + "\x00" + errs + "\x00"))
		} else {
			var stat interface {}
			if err == nil {
				stat = "OK."
			} else {
				stat = err
			}
			fmt.Println(file + ":", stat)
		}
	}()
	if err != nil { return }

	srcf0, err := os.Open(file)
	if err != nil { return }
	d, err := t.Delta(bufio.NewReader(srcf0))
	if err != nil { return }

	if (*debug) {
		fmt.Fprintln(os.Stderr, file + ":")
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

	fi, err := srcf0.Stat()
	if err != nil { return }
	var x interface {}
	err = cli.Call("Server.Patch",
	               &rsync.PatchArg{Basep: file,
	                               Delta: d,
	                               Uid: fi.Uid, Gid: fi.Gid,
	                               Permission: fi.Permission()},
	               &x)
}

const (
	SND = "SEND"
	RCV = "RECEIVE"
)

func main() {
	gob.Register(rsync.Byterange{})

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] <remote-cmd-with-args> [<file> ...] %s <base>\n", os.Args[0], SND)
		fmt.Fprintf(os.Stderr, "%s [options] <remote-cmd> [<remote-args> ...] <argsep> [<file> ...] %s <base>\n", os.Args[0], SND)
		fmt.Fprintf(os.Stderr, "%s [options] %s <base>\n", os.Args[0], RCV)
		fmt.Fprintln(os.Stderr, "Options can be:")
		flag.PrintDefaults()
	}
	blocksize := flag.Int("blocksize", 4096, "blocksize used in rsync algorithm")
	zerosep = flag.Bool("0", false, "separate file names by zero byte")
	argsep := flag.String("argsep", "--", "string to separate remote command from file args")
	debug = flag.Bool("debug", false, "debug mode")
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		flag.Usage()
		os.Exit(2)
	}

	log.SetPrefix(args[len(args) - 2] + " ")
	err := os.Chdir(args[len(args) - 1])
	if err != nil { log.Fatal(err) }

	switch args[len(args) - 2] {
	case RCV:
		rpc.Register(rsync.NewServer(*blocksize))
		rsync.Serve(os.Stdin, os.Stdout)

		return
	case SND:
	default:
		log.Fatal("unkown mode")
	}

	args = args[:len(args) - 2]
	if len(args) < 1 {
		flag.Usage()
		os.Exit(2)
	}

	var remcmdargs, files []string
	for i, arg := range args {
		if arg == *argsep {
			remcmdargs = args[:i]
			files = args[i+1:]
			break
		}
	}

	if remcmdargs == nil {
		remcmdargs0 := strings.Split(args[0], " ")
		//collapse entries coming from subsequent spaces
		for _, a := range(remcmdargs0) {
			if len(a) != 0 {
				remcmdargs = append(remcmdargs, a)
			}
		}
		files = args[1:]
	}

	remcmd := exec.Command(remcmdargs[0], remcmdargs[1:]...)
	remin, err := remcmd.StdinPipe()
	if err != nil { log.Fatal(err) }
	remout, err := remcmd.StdoutPipe()
	if err != nil { log.Fatal(err) }
	remcmd.Stderr = os.Stderr
	err = remcmd.Start()
	if err != nil { log.Fatal(err) }

	go func() {
		err := remcmd.Wait()
		log.Fatal(fmt.Sprint("remote end hang up with error ", err))
	}()

	client := rsync.Client(remout, remin)

	var wg sync.WaitGroup
	syncFile_bg := func(cli *rpc.Client, file string) {
		wg.Add(1)
		go func() {
			syncFile(cli, file)
			wg.Done()
		}()
	}

	if len(files) == 0 {
		var sepchar byte
		sepchar = '\n'
		if *zerosep { sepchar = '\x00' }
		r := bufio.NewReader(os.Stdin)
		for {
			fp, err := r.ReadString(sepchar)
			if err == os.EOF { break }
			if err != nil { log.Fatal(err) }
			syncFile_bg(client, fp[:len(fp)-1])
		}
	} else {
		for _, fp := range(files) { syncFile_bg(client, fp) }
	}

	wg.Wait()
}
