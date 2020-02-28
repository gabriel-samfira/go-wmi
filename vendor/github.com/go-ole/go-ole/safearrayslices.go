// +build windows

package ole

import (
	"unsafe"
)

func safeArrayFromByteSlice(slice []byte) *SafeArray {
	array, _ := safeArrayCreateVector(VT_UI1, 0, uint32(len(slice)))

	if array == nil {
		panic("Could not convert []byte to SAFEARRAY")
	}

	for i, v := range slice {
		safeArrayPutElement(array, int64(i), uintptr(unsafe.Pointer(&v)))
	}
	return array
}

func safeArrayFromStringSlice(slice []string) *SafeArray {
	array, _ := safeArrayCreateVector(VT_BSTR, 0, uint32(len(slice)))

	if array == nil {
		panic("Could not convert []string to SAFEARRAY")
	}
	// SysAllocStringLen(s)
	for i, v := range slice {
		safeArrayPutElement(array, int64(i), uintptr(unsafe.Pointer(SysAllocStringLen(v))))
	}
	return array
}

func safeArrayFromUintSlice(slice []uint16) *SafeArray {
	array, _ := safeArrayCreateVector(VT_UI2, 0, uint32(len(slice)))

	if array == nil {
		panic("Could not convert []uint16 to SAFEARRAY")
	}

	for i, v := range slice {
		safeArrayPutElement(array, int64(i), uintptr(unsafe.Pointer(&v)))
	}
	return array
}

func safeArrayFromIntSlice(slice []int16) *SafeArray {
	array, _ := safeArrayCreateVector(VT_I2, 0, uint32(len(slice)))

	if array == nil {
		panic("Could not convert []int16 to SAFEARRAY")
	}

	for i, v := range slice {
		safeArrayPutElement(array, int64(i), uintptr(unsafe.Pointer(&v)))
	}
	return array
}

func safeArrayFromInt32Slice(slice []int32) *SafeArray {
	array, _ := safeArrayCreateVector(VT_I4, 0, uint32(len(slice)))

	if array == nil {
		panic("Could not convert []int32 to SAFEARRAY")
	}

	for i, v := range slice {
		safeArrayPutElement(array, int64(i), uintptr(unsafe.Pointer(&v)))
	}
	return array
}
