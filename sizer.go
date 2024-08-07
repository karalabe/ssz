// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import "github.com/prysmaticlabs/go-bitfield"

// Sizer is an SSZ static and dynamic size computer.
type Sizer struct {
	codec *Codec // Self-referencing to have access to fork contexts
}

// Fork retrieves the current fork (if any) that the sizer is operating in.
func (siz *Sizer) Fork() Fork {
	return siz.codec.fork
}

// SizeDynamicBytes returns the serialized size of the dynamic part of a dynamic
// blob.
func SizeDynamicBytes(siz *Sizer, blobs []byte) uint32 {
	return uint32(len(blobs))
}

// SizeSliceOfBits returns the serialized size of the dynamic part of a slice of
// bits.
func SizeSliceOfBits(siz *Sizer, bits bitfield.Bitlist) uint32 {
	return uint32(len(bits))
}

// SizeSliceOfUint64s returns the serialized size of the dynamic part of a dynamic
// list of uint64s.
func SizeSliceOfUint64s[T ~uint64](siz *Sizer, ns []T) uint32 {
	return uint32(len(ns)) * 8
}

// SizeDynamicObject returns the serialized size of the dynamic part of a dynamic
// object.
func SizeDynamicObject[T DynamicObject](siz *Sizer, obj T) uint32 {
	return obj.SizeSSZ(siz, false)
}

// SizeSliceOfStaticBytes returns the serialized size of the dynamic part of a dynamic
// list of static blobs.
func SizeSliceOfStaticBytes[T commonBytesLengths](siz *Sizer, blobs []T) uint32 {
	if len(blobs) == 0 {
		return 0
	}
	return uint32(len(blobs) * len(blobs[0]))
}

// SizeSliceOfDynamicBytes returns the serialized size of the dynamic part of a dynamic
// list of dynamic blobs.
func SizeSliceOfDynamicBytes(siz *Sizer, blobs [][]byte) uint32 {
	var size uint32
	for _, blob := range blobs {
		size += uint32(4 + len(blob)) // 4-byte offset + dynamic data later
	}
	return size
}

// SizeSliceOfStaticObjects returns the serialized size of the dynamic part of a dynamic
// list of static objects.
func SizeSliceOfStaticObjects[T StaticObject](siz *Sizer, objects []T) uint32 {
	if len(objects) == 0 {
		return 0
	}
	return uint32(len(objects)) * objects[0].SizeSSZ(siz)
}

// SizeSliceOfDynamicObjects returns the serialized size of the dynamic part of
// a dynamic list of dynamic objects.
func SizeSliceOfDynamicObjects[T DynamicObject](siz *Sizer, objects []T) uint32 {
	var size uint32
	for _, obj := range objects {
		size += 4 + obj.SizeSSZ(siz, false) // 4-byte offset + dynamic data later
	}
	return size
}
