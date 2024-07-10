// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

// newableStaticObject is a generic type whose purpose is to enforce that the
// ssz.StaticObject is specifically implemented on a struct pointer. That is
// needed to allow to instantiate new structs via `new` when parsing.
type newableStaticObject[U any] interface {
	StaticObject
	*U
}

// newableDynamicObject is a generic type whose purpose is to enforce that the
// ssz.DynamicObject is specifically implemented on a struct pointer. That is
// needed to allow to instantiate new structs via `new` when parsing.
type newableDynamicObject[U any] interface {
	DynamicObject
	*U
}

// commonBytesLengths is a generic type whose purpose is to permit that fixed-
// sized binary blobs can be passed to different methods. Although a slice of
// the array would work for simple cases, there are scenarios when a new array
// needs to be instantiated (e.g. optional field), and in that case we cannot
// operate on slices anymore as there's nothing yet to slice.
//
// You can add any size to this list really, it's just a limitation of the Go
// generics compiler that it cannot represent arrays of arbitrary sizes with
// one shorthand notation.
type commonBytesLengths interface {
	// fork | address | verkle-stem | hash | pubkey | committee | signature | bloom
	~[4]byte | ~[20]byte | ~[31]byte | ~[32]byte | ~[48]byte | ~[64]byte | ~[96]byte | ~[256]byte
}

// commonUint64sLengths is a generic type whose purpose is to permit that fixed-
// sized uint64 arrays can be passed to different methods. Although a slice of
// the array would work for simple cases, there are scenarios when a new array
// needs to be instantiated (e.g. optional field), and in that case we cannot
// operate on slices anymore as there's nothing yet to slice.
//
// You can add any size to this list really, it's just a limitation of the Go
// generics compiler that it cannot represent arrays of arbitrary sizes with
// one shorthand notation.
type commonUint64sLengths interface {
	// slashing
	~[8192]uint64
}

// commonBitsLengths is a generic type whose purpose is to permit that fixed-
// sized bit-vectors can be passed to different methods. Although a slice of
// the array would work for simple cases, there are scenarios when a new array
// needs to be instantiated (e.g. optional field), and in that case we cannot
// operate on slices anymore as there's nothing yet to slice.
//
// You can add any size to this list really, it's just a limitation of the Go
// generics compiler that it cannot represent arrays of arbitrary sizes with
// one shorthand notation.
type commonBitsLengths interface {
	// justification
	~[1]byte
}

// commonBytesArrayLengths is a generic type whose purpose is to permit that
// lists of different fixed-sized binary blob arrays can be passed to methods.
//
// You can add any size to this list really, it's just a limitation of the Go
// generics compiler that it cannot represent arrays of arbitrary sizes with
// one shorthand notation.
type commonBytesArrayLengths[U commonBytesLengths] interface {
	// proof | committee | history | randao
	~[33]U | ~[512]U | ~[8192]U | ~[65536]U
}
