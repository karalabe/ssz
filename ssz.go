// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

// Package ssz is a simplified SSZ encoder/decoder.
package ssz

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

// Object defines the methods a type needs to implement to be used as a ssz
// encodable and decodable object.
type Object interface {
	// DefineSSZ defines how an object would be encoded/decoded.
	DefineSSZ(codec *Codec)
}

// StaticObject defines the methods a type needs to implement to be used as a
// ssz encodable and decodable static object.
type StaticObject interface {
	Object

	// SizeSSZ returns the total size of the ssz object.
	//
	// Note, StaticObject.SizeSSZ and DynamicObject.SizeSSZ deliberately clash
	// to allow the compiler to detect placing one or the other in reversed data
	// slots on an SSZ containers.
	SizeSSZ() uint32
}

// DynamicObject defines the methods a type needs to implement to be used as a
// ssz encodable and decodable dynamic object.
type DynamicObject interface {
	Object

	// SizeSSZ returns either the static size of the object if fixed == true, or
	// the total size otherwise.
	//
	// Note, StaticObject.SizeSSZ and DynamicObject.SizeSSZ deliberately clash
	// to allow the compiler to detect placing one or the other in reversed data
	// slots on an SSZ containers.
	SizeSSZ(fixed bool) uint32
}

// encoderPool is a pool of SSZ encoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var encoderPool = sync.Pool{
	New: func() any {
		codec := &Codec{enc: new(Encoder)}
		codec.enc.codec = codec
		return codec
	},
}

// decoderPool is a pool of SSZ decoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var decoderPool = sync.Pool{
	New: func() any {
		codec := &Codec{dec: new(Decoder)}
		codec.dec.codec = codec
		return codec
	},
}

// Encode serializes the object into an SSZ stream.
func Encode(w io.Writer, obj Object) error {
	codec := encoderPool.Get().(*Codec)
	defer encoderPool.Put(codec)

	codec.enc.out, codec.enc.err = w, nil
	obj.DefineSSZ(codec)
	return codec.enc.err
}

// EncodeToBytes serializes the object into a newly allocated byte buffer.
func EncodeToBytes(obj Object) ([]byte, error) {
	buffer := make([]byte, Size(obj))
	if err := Encode(bytes.NewBuffer(buffer[:0]), obj); err != nil {
		return nil, err
	}
	return buffer, nil
}

// Decode parses an object with the given size out of an SSZ stream.
func Decode(r io.Reader, obj Object, size uint32) error {
	codec := decoderPool.Get().(*Codec)
	defer decoderPool.Put(codec)

	codec.dec.in, codec.dec.length, codec.dec.err = r, size, nil
	obj.DefineSSZ(codec)
	return codec.dec.err
}

// DecodeFromBytes parses an object from the given SSZ blob.
func DecodeFromBytes(blob []byte, obj Object) error {
	return Decode(bytes.NewReader(blob), obj, uint32(len(blob)))
}

// Size retrieves the size of a ssz object, independent if it's a static or a
// dynamic one.
func Size(obj Object) uint32 {
	var size uint32
	switch v := obj.(type) {
	case StaticObject:
		size = v.SizeSSZ()
	case DynamicObject:
		size = v.SizeSSZ(false)
	default:
		panic(fmt.Sprintf("unsupported type: %T", obj))
	}
	return size
}
