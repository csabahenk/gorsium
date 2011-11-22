package md5

// #include <compat.h>
// #include <mdigest.h>
// #include <md5_2.h>
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

func Md5_2(b1 []byte, b2 []byte) []byte {
	s := make([]byte, 16)

	sh := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	bh1 := (*reflect.SliceHeader)(unsafe.Pointer(&b1))
	bh2 := (*reflect.SliceHeader)(unsafe.Pointer(&b2))

	C.get_md5_2((*C.uchar)(unsafe.Pointer(sh.Data)), (*C.uchar)(unsafe.Pointer(bh1.Data)), C.int(len(b1)),
		    (*C.uchar)(unsafe.Pointer(bh2.Data)), C.int(len(b2)))

	return s
}
