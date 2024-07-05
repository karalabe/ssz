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

// commonBinaryLengths is a generic type whose purpose is to permit that lists
// of different fixed-sized binary blobs can be passed to methods.
//
// You can add any size to this list really, it's just a limitation of the Go
// generics compiler that it cannot represent arrays of arbitrary sizes with
// one shorthand notation.
type commonBinaryLengths interface {
	// footgun | address | hash | pubkey | signature
	~[]byte | ~[20]byte | ~[32]byte | [48]byte | [96]byte
}
