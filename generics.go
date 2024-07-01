// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

// newableObject is a generic type whose purpose is to enforce that ssz.Object
// is specifically implemented on a struct pointer. That's needed to allow us
// to instantiate new structs via `new` when parsing.
type newableObject[U any] interface {
	Object
	*U
}

// commonBinaryLengths is a generic type whose purpose is to permit that lists
// of different fixed-sized binary blobs can be passed to methods.
//
// You can add any size to this list really, it's just a limitation of the Go
// generics compiler that it cannot represent arrays of arbitrary sizes with
// one shorthand notation.
type commonBinaryLengths interface {
	// footgun | address | hash | pubkey
	~[]byte | ~[20]byte | ~[32]byte | [48]byte
}
