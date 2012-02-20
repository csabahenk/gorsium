package rsync

import (
	"bufio"
	"os"
	"path"
	"io"
	"io/ioutil"
	"rpc"
)


// pipe pair based rpc

type pipeRwc struct {
	rpipe io.ReadCloser
	wpipe io.WriteCloser
}

func newpipeRwc(inf io.ReadCloser, ouf io.WriteCloser) *pipeRwc {
	return &pipeRwc{inf, ouf}
}

func (prwc *pipeRwc) Read(b []byte) (n int, err os.Error) {
	return prwc.rpipe.Read(b)
}

func (prwc *pipeRwc) Write(b []byte) (n int, err os.Error) {
	return prwc.wpipe.Write(b)
}

func (prwc *pipeRwc) closeRead() os.Error {
	return prwc.rpipe.Close()
}

func (prwc *pipeRwc) closeWrite() os.Error {
	return prwc.wpipe.Close()
}

func (prwc *pipeRwc) Close() os.Error {
	e1 := prwc.closeRead()
	e2 := prwc.closeWrite()
	if e1 != nil { return e1 }
	return e2
}

func Serve(in io.ReadCloser, out io.WriteCloser) {
	rpc.DefaultServer.ServeConn(newpipeRwc(in, out))
}

func Client(in io.ReadCloser, out io.WriteCloser) *rpc.Client {
	return rpc.NewClient(newpipeRwc(in, out))
}


// rsync server

type Server struct {
	blocksize int
	fdtable map[string] *os.File
}

func NewServer(blocksize int) *Server {
	s := Server{blocksize: blocksize}
	s.fdtable = make(map [string] *os.File)

	return &s
}

func checkPath(pname string) os.Error {
	if path.IsAbs(pname) { return os.NewError("invalid path") }
	for i := -1; i < len(pname) - 3; i++ {
		if (i == -1 || pname[i] == os.PathSeparator) &&
		   pname[i+1] == '.' && pname[i+2] == '.' &&
		   (i == len(pname) - 3 || pname[i + 3] == os.PathSeparator) {
			return os.NewError("invalid path")
		}
	}

	return nil
}

func (s *Server) Sumtable(pname string, tp **SumTable) os.Error {
	err := checkPath(pname)
	if err != nil { return err }

	var f io.Reader
	f0, err := os.Open(pname)
	if err == nil {
		f, err = bufio.NewReaderSize(f0, s.blocksize)
	} else if perr, ok := err.(*os.PathError); ok && perr.Error == os.ENOENT {
		f0 = nil
		err = nil
	}
	if err != nil { return err }

	t, err := SumTableOf(f, s.blocksize)
	if err != nil { return err }
	*tp = t
	s.fdtable[pname] = f0

	return nil
}

type PatchArg struct {
	Basep string
	Delta Delta
	Uid, Gid int
	Permission uint32
}

func (s *Server) Patch(pa *PatchArg, x *interface{}) os.Error {
	f, ok := s.fdtable[pa.Basep]
	if !ok { return os.NewError("base file not in registry") }
	defer func() {
		s.fdtable[pa.Basep] = nil, false
		if f != nil { f.Close() }
	}()

	tdir, tname := path.Split(pa.Basep)
	if tdir == "" {
		tdir = "."
	}
	tgtf, err := ioutil.TempFile(tdir, tname + ".")
	if err != nil { return err }
	defer tgtf.Close()
	err = Patch(tgtf, f, &pa.Delta)
	if err == nil { err = tgtf.Chmod(pa.Permission) }
	if err == nil { err = tgtf.Chown(pa.Uid, pa.Gid) }
	if err == nil {
		err = os.Rename(tgtf.Name(), pa.Basep)
	} else {
		os.Remove(tgtf.Name())
	}

	return err
}
