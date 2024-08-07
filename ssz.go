// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

// Package ssz is a simplified SSZ encoder/decoder.
package ssz

import (
	"fmt"
	"io"
	"sync"
	"unsafe"
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

// hasherPool is a pool of SSZ hashers to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var hasherPool = sync.Pool{
	New: func() any {
		codec := &Codec{has: new(Hasher)}
		codec.has.codec = codec
		return codec
	},
}

// treererPool is a pool of SSZ hashers to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var treererPool = sync.Pool{
	New: func() any {
		codec := &Codec{tre: new(Treerer), has: new(Hasher), enc: new(Encoder)}
		codec.has.codec = codec
		codec.tre.codec = codec
		codec.enc.codec = codec

		return codec
	},
}

// EncodeToStream serializes the object into a data stream. Do not use this
// method with a bytes.Buffer to write into a []byte slice, as that will do
// double the byte copying. For that use case, use EncodeToBytes instead.
func EncodeToStream(w io.Writer, obj Object) error {
	codec := encoderPool.Get().(*Codec)
	defer encoderPool.Put(codec)

	codec.enc.outWriter, codec.enc.err = w, nil
	switch v := obj.(type) {
	case StaticObject:
		v.DefineSSZ(codec)
	case DynamicObject:
		codec.enc.offsetDynamics(v.SizeSSZ(true))
		v.DefineSSZ(codec)
	default:
		panic(fmt.Sprintf("unsupported type: %T", obj))
	}
	codec.enc.outWriter = nil
	return codec.enc.err
}

// EncodeToBytes serializes the object into a byte buffer. Don't use this method
// if you want to then write the buffer into a stream via some writer, as that
// would double the memory use for the temporary buffer. For that use case, use
// EncodeToStream instead.
func EncodeToBytes(buf []byte, obj Object) error {
	// Sanity check that we have enough space to serialize into
	if size := Size(obj); int(size) > len(buf) {
		return fmt.Errorf("%w: buffer %d bytes, object %d bytes", ErrBufferTooSmall, len(buf), size)
	}
	codec := encoderPool.Get().(*Codec)
	defer encoderPool.Put(codec)

	codec.enc.outBuffer, codec.enc.err = buf, nil
	switch v := obj.(type) {
	case StaticObject:
		v.DefineSSZ(codec)
	case DynamicObject:
		codec.enc.offsetDynamics(v.SizeSSZ(true))
		v.DefineSSZ(codec)
	default:
		panic(fmt.Sprintf("unsupported type: %T", obj))
	}
	codec.enc.outBuffer = nil
	return codec.enc.err
}

// DecodeFromStream parses an object with the given size out of a stream. Do not
// use this method with a bytes.Buffer to read from a []byte slice, as that will
// double the byte copying. For that use case, use DecodeFromBytes instead.
func DecodeFromStream(r io.Reader, obj Object, size uint32) error {
	// Retrieve a new decoder codec and set its data source
	codec := decoderPool.Get().(*Codec)
	defer decoderPool.Put(codec)

	codec.dec.inReader = r

	// Start a decoding round with length enforcement in place
	codec.dec.descendIntoSlot(size)

	switch v := obj.(type) {
	case StaticObject:
		v.DefineSSZ(codec)
	case DynamicObject:
		codec.dec.startDynamics(v.SizeSSZ(true))
		v.DefineSSZ(codec)
		codec.dec.flushDynamics()
	default:
		panic(fmt.Sprintf("unsupported type: %T", obj))
	}
	codec.dec.ascendFromSlot()

	// Retrieve any errors, zero out the source and return
	err := codec.dec.err

	codec.dec.inReader = nil
	codec.dec.err = nil

	return err
}

// DecodeFromBytes parses an object from a byte buffer. Do not use this method
// if you want to first read the buffer from a stream via some reader, as that
// would double the memory use for the temporary buffer. For that use case, use
// DecodeFromStream instead.
func DecodeFromBytes(blob []byte, obj Object) error {
	// Reject decoding from an empty slice
	if len(blob) == 0 {
		return io.ErrUnexpectedEOF
	}
	// Retrieve a new decoder codec and set its data source
	codec := decoderPool.Get().(*Codec)
	defer decoderPool.Put(codec)

	codec.dec.inBuffer = blob
	codec.dec.inBufEnd = uintptr(unsafe.Pointer(&blob[0])) + uintptr(len(blob))

	// Start a decoding round with length enforcement in place
	codec.dec.descendIntoSlot(uint32(len(blob)))

	switch v := obj.(type) {
	case StaticObject:
		v.DefineSSZ(codec)
	case DynamicObject:
		codec.dec.startDynamics(v.SizeSSZ(true))
		v.DefineSSZ(codec)
		codec.dec.flushDynamics()
	default:
		panic(fmt.Sprintf("unsupported type: %T", obj))
	}
	codec.dec.ascendFromSlot()

	// Retrieve any errors, zero out the source and return
	err := codec.dec.err

	codec.dec.inBufEnd = 0
	codec.dec.inBuffer = nil
	codec.dec.err = nil

	return err
}

// HashSequential computes the ssz merkle root of the object on a single thread.
// This is useful for processing small objects with stable runtime and O(1) GC
// guarantees.
func HashSequential(obj Object) [32]byte {
	codec := hasherPool.Get().(*Codec)
	defer hasherPool.Put(codec)
	defer codec.has.Reset()

	codec.has.descendLayer()
	obj.DefineSSZ(codec)
	codec.has.ascendLayer(0)

	if len(codec.has.chunks) != 1 {
		panic(fmt.Sprintf("unfinished hashing: left %v", codec.has.groups))
	}
	return codec.has.chunks[0]
}

// HashConcurrent computes the ssz merkle root of the object on potentially multiple
// concurrent threads (iff some data segments are large enough to be worth it). This
// is useful for processing large objects, but will place a bigger load on your CPU
// and GC; and might be more variable timing wise depending on other load.
func HashConcurrent(obj Object) [32]byte {
	codec := hasherPool.Get().(*Codec)
	defer hasherPool.Put(codec)
	defer codec.has.Reset()

	codec.has.threads = true
	codec.has.descendLayer()
	obj.DefineSSZ(codec)
	codec.has.ascendLayer(0)

	if len(codec.has.chunks) != 1 {
		panic(fmt.Sprintf("unfinished hashing: left %v", codec.has.groups))
	}
	return codec.has.chunks[0]
}

// TreeSequential computes the ssz merkle tree of the object on a single thread.
// This is useful for processing small objects with stable runtime and O(1) GC
// guarantees.
func TreeSequential(obj Object) *TreeNode {
	codec := treererPool.Get().(*Codec)
	defer treererPool.Put(codec)
	defer codec.tre.Reset()
	obj.DefineSSZ(codec)
	return codec.tre.GetRoot()
}

// TreeConcurrent computes the SSZ Merkle tree of the object on a multiple threads.
func TreeConcurrent(obj Object) *TreeNode {
	panic("not implemented yet")
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
