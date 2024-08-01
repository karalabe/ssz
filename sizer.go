// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import "github.com/prysmaticlabs/go-bitfield"

// SizeDynamicBytes returns the serialized size of the dynamic part of a dynamic
// blob.
func SizeDynamicBytes(blobs []byte) uint32 {
	return uint32(len(blobs))
}

// SizeSliceOfBits returns the serialized size of the dynamic part of a slice of
// bits.
func SizeSliceOfBits(bits bitfield.Bitlist) uint32 {
	return uint32(len(bits))
}

// SizeSliceOfUint64s returns the serialized size of the dynamic part of a dynamic
// list of uint64s.
func SizeSliceOfUint64s[T ~uint64](ns []T) uint32 {
	return uint32(len(ns)) * 8
}

// SizeDynamicObject returns the serialized size of the dynamic part of a dynamic
// object.
func SizeDynamicObject[T DynamicObjectSizer](obj T) uint32 {
	return obj.SizeSSZ(false)
}

// SizeSliceOfStaticBytes returns the serialized size of the dynamic part of a dynamic
// list of static blobs.
func SizeSliceOfStaticBytes[T commonBytesLengths](blobs []T) uint32 {
	if len(blobs) == 0 {
		return 0
	}
	return uint32(len(blobs) * len(blobs[0]))
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
func SizeSliceOfStaticObjects[T StaticObjectSizer](objects []T) uint32 {
	if len(objects) == 0 {
		return 0
	}
	return uint32(len(objects)) * objects[0].SizeSSZ()
}

// SizeSliceOfDynamicObjects returns the serialized size of the dynamic part of
// a dynamic list of dynamic objects.
func SizeSliceOfDynamicObjects[T DynamicObjectSizer](objects []T) uint32 {
	var size uint32
	for _, obj := range objects {
		size += 4 + obj.SizeSSZ(false) // 4-byte offset + dynamic data later
	}
	return size
}
