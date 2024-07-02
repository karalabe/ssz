// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

// Package ssz contains a few coding helpers to implement SSZ codecs.
package ssz

import (
	"fmt"
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

	// DefineSSZ runs the object's schema definition against an SSZ codec.
	DefineSSZ(codec *Codec)
}

// encoderPool is a pool of SSZ encoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var encoderPool = sync.Pool{
	New: func() any {
		return &Codec{enc: new(Encoder)}
	},
}

// decoderPool is a pool of SSZ edecoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var decoderPool = sync.Pool{
	New: func() any {
		return &Codec{dec: new(Decoder)}
	},
}

// Encode serializes the provided object into an SSZ stream.
func Encode(w io.Writer, obj Object) error {
	codec := encoderPool.Get().(*Codec)
	defer encoderPool.Put(codec)

	codec.enc.out, codec.enc.err, codec.enc.dyn = w, nil, false
	obj.DefineSSZ(codec)

	if codec.enc.err == nil && codec.enc.dyn && obj.StaticSSZ() {
		return fmt.Errorf("%w: %T", ErrStaticObjectBehavedDynamic, obj)
	}
	return codec.enc.err
}

// Decode parses an object with the given size out of an SSZ stream.
func Decode(r io.Reader, obj Object, size uint32) error {
	codec := decoderPool.Get().(*Codec)
	defer decoderPool.Put(codec)

	codec.dec.in, codec.dec.length, codec.dec.err, codec.dec.dyn = r, size, nil, false
	obj.DefineSSZ(codec)

	if codec.dec.err == nil && codec.dec.dyn && obj.StaticSSZ() {
		return fmt.Errorf("%w: %T", ErrStaticObjectBehavedDynamic, obj)
	}
	return codec.dec.err
}
