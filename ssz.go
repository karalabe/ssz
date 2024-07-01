// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

// Package ssz contains a few coding helpers to implement SSZ codecs.
package ssz

import (
	"io"
	"sync"
)

// Object defines the methods a type needs to implement to be used as an SSZ
// encodable and decodable object.
type Object interface {
	// StaticSSZ returns whether the object is static in size (i.e. always takes
	// up the same space to encode) or variable.
	//
	// Note, this method *must* be implemented on the pointer type and should
	// simply return true or false. It *will* be called on nil.
	StaticSSZ() bool

	// SizeSSZ returns the total size of an SSZ object.
	SizeSSZ() uint32

	// EncodeSSZ serializes the object though an SSZ encoder.
	EncodeSSZ(enc *Encoder)

	// DecodeSSZ parses the object via an SSZ decoder.
	DecodeSSZ(dec *Decoder)
}

// encoderPool is a pool of SSZ encoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var encoderPool = sync.Pool{
	New: func() any {
		return new(Encoder)
	},
}

// decoderPool is a pool of SSZ edecoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var decoderPool = sync.Pool{
	New: func() any {
		return new(Decoder)
	},
}

// Encode serializes the provided object into an SSZ stream.
func Encode(w io.Writer, obj Object) error {
	enc := encoderPool.Get().(*Encoder)
	defer encoderPool.Put(enc)

	enc.out, enc.err = w, nil
	obj.EncodeSSZ(enc)
	return enc.err
}

// Decode parses an object with the given size out of an SSZ stream.
func Decode(r io.Reader, obj Object, size uint32) error {
	dec := decoderPool.Get().(*Decoder)
	defer decoderPool.Put(dec)

	dec.in, dec.length, dec.err = r, size, nil
	obj.DecodeSSZ(dec)
	return dec.err
}
