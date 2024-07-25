// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import "fmt"

// PrecomputeStaticSizeCache is a helper for genssz to precompute SSZ (static)
// sizes for a monolith type on different forks.
//
// For non-monolith types that are constant across forks (or are not meant to be
// used across forks), all the sizes will be the same.
func PrecomputeStaticSizeCache(obj Object) []uint32 {
	var (
		sizes = make([]uint32, ForkFuture)
		sizer = &Sizer{codec: new(Codec)}
	)
	switch v := obj.(type) {
	case StaticObject:
		for fork := 0; fork < len(sizes); fork++ {
			sizer.codec.fork = Fork(fork)
			sizes[fork] = v.SizeSSZ(sizer)
		}
	case DynamicObject:
		for fork := 0; fork < len(sizes); fork++ {
			sizer.codec.fork = Fork(fork)
			sizes[fork] = v.SizeSSZ(sizer, true)
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", obj))
	}
	return sizes
}
