// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

// SizeDynamicBytes returns the serialized size of the dynamic part of a dynamic
// blob.
func SizeDynamicBytes(blobs []byte) uint32 {
	return uint32(len(blobs))
}

// SizeSliceOfUint64s returns the serialized size of the dynamic part of a dynamic
// list of uint64s.
func SizeSliceOfUint64s[T ~uint64](ns []T) uint32 {
	return uint32(len(ns)) * 8
}

// SizeDynamicObject returns the serialized size of the dynamic part of a dynamic
// object.
func SizeDynamicObject(obj Object) uint32 {
	return obj.SizeSSZ()
}

// SizeSliceOfDynamicBytes returns the serialized size of the dynamic part of a dynamic
// list of dynamic blobs.
func SizeSliceOfDynamicBytes(blobs [][]byte) uint32 {
	var size uint32
	for _, blob := range blobs {
		size += uint32(4 + len(blob)) // 4-byte offset + dynamic data later
	}
	return size
}

// SizeSliceOfStaticObjects returns the serialized size of the dynamic part of a dynamic
// list of static objects.
func SizeSliceOfStaticObjects[T Object](objects []T) uint32 {
	if len(objects) == 0 {
		return 0
	}
	return uint32(len(objects)) * objects[0].SizeSSZ()
}
