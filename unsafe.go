package gotiny

import (
	"reflect"
	"unsafe"
)

type refVal struct {
	_    unsafe.Pointer
	ptr  unsafe.Pointer
	flag uintptr
}

const flagIndir uintptr = 1 << 7

// getUnsafePointer returns an unsafe.Pointer to the data held by the given reflect.Value.
// If the reflect.Value is not indirect, it returns a pointer to the value itself.
// Otherwise, it returns the pointer stored in the reflect.Value.
//
// Parameters:
//   - rv: The reflect.Value from which to extract the unsafe.Pointer.
//
// Returns:
//   - unsafe.Pointer: A pointer to the data held by the reflect.Value.
func getUnsafePointer(rv reflect.Value) unsafe.Pointer {
	vv := (*refVal)(unsafe.Pointer(&rv))
	if vv.flag&flagIndir == 0 {
		return unsafe.Pointer(&vv.ptr)
	} else {
		return vv.ptr
	}
}

type sliceHeader struct {
	data unsafe.Pointer
	len  int
	cap  int
}
