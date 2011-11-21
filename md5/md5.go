package md5

// #include <compat.h>
// #include <mdigest.h>
import "C"
import "unsafe"
import "reflect"

func Md5(b []byte) []byte {
	s := make([]byte, 16)

	sh := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))

	C.get_md5((*C.uchar)(unsafe.Pointer(sh.Data)), (*C.uchar)(unsafe.Pointer(bh.Data)), C.int(len(b)))

	return s
}
