// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

// SizeDynamicBlob returns the serialized size of the dynamic part of a dynamic
// blob.
func SizeDynamicBlob(blob []byte) uint32 {
	return uint32(len(blob))
}

// SizeDynamicBlobs returns the serialized size of the dynamic part of a dynamic
// list of dynamic blobs.
func SizeDynamicBlobs(blobs [][]byte) uint32 {
	var size uint32
	for _, blob := range blobs {
		size += uint32(4 + len(blob)) // 4-byte offset + dynamic data later
	}
	return size
}

// SizeDynamicStatics returns the serialized size of the dynamic part of a dynamic
// list of static objects.
func SizeDynamicStatics[T Object](objects []T) uint32 {
	if len(objects) == 0 {
		return 0
	}
	return uint32(len(objects)) * objects[0].SizeSSZ()
}
