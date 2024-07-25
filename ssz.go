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
	SizeSSZ(siz *Sizer) uint32
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
	SizeSSZ(siz *Sizer, fixed bool) uint32
}

// encoderPool is a pool of SSZ encoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var encoderPool = sync.Pool{
	New: func() any {
		codec := &Codec{enc: new(Encoder)}
		codec.enc.codec = codec
		codec.enc.sizer = &Sizer{codec: codec}
		return codec
	},
}

// decoderPool is a pool of SSZ decoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var decoderPool = sync.Pool{
	New: func() any {
		codec := &Codec{dec: new(Decoder)}
		codec.dec.codec = codec
		codec.dec.sizer = &Sizer{codec: codec}
		return codec
	},
}

// hasherPool is a pool of SSZ hashers to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var hasherPool = sync.Pool{
	New: func() any {
		codec := &Codec{has: new(Hasher)}
		codec.has.codec = codec
		codec.has.sizer = &Sizer{codec: codec}
		return codec
	},
}

// sizerPool is a pool of SSZ sizers to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var sizerPool = sync.Pool{
	New: func() any {
		return &Sizer{codec: new(Codec)}
	},
}

// EncodeToStream serializes the object into a data stream. Do not use this
// method with a bytes.Buffer to write into a []byte slice, as that will do
// double the byte copying. For that use case, use EncodeToBytes instead.
func EncodeToStream(w io.Writer, obj Object) error {
	return EncodeToStreamWithFork(w, obj, ForkUnknown)
}

// EncodeToStreamWithFork is analogous to EncodeToStream, but allows the user to
// set a specific fork context to encode the object in. This is useful for code-
// bases that have monolith types that marshal into many fork formats.
func EncodeToStreamWithFork(w io.Writer, obj Object, fork Fork) error {
	codec := encoderPool.Get().(*Codec)
	defer encoderPool.Put(codec)

	codec.fork, codec.enc.outWriter = fork, w
	switch v := obj.(type) {
	case StaticObject:
		v.DefineSSZ(codec)
	case DynamicObject:
		codec.enc.offsetDynamics(v.SizeSSZ(codec.enc.sizer, true))
		v.DefineSSZ(codec)
	default:
		panic(fmt.Sprintf("unsupported type: %T", obj))
	}
	// Retrieve any errors, zero out the sink and return
	err := codec.enc.err

	codec.enc.outWriter = nil
	codec.enc.err = nil

	return err
}

// EncodeToBytes serializes the object into a byte buffer. Don't use this method
// if you want to then write the buffer into a stream via some writer, as that
// would double the memory use for the temporary buffer. For that use case, use
// EncodeToStream instead.
func EncodeToBytes(buf []byte, obj Object) error {
	return EncodeToBytesWithFork(buf, obj, ForkUnknown)
}

// EncodeToBytesWithFork is analogous to EncodeToBytes, but allows the user to
// set a specific fork context to encode the object in. This is useful for code-
// bases that have monolith types that marshal into many fork formats.
func EncodeToBytesWithFork(buf []byte, obj Object, fork Fork) error {
	// Sanity check that we have enough space to serialize into
	if size := Size(obj); int(size) > len(buf) {
		return fmt.Errorf("%w: buffer %d bytes, object %d bytes", ErrBufferTooSmall, len(buf), size)
	}
	codec := encoderPool.Get().(*Codec)
	defer encoderPool.Put(codec)

	codec.fork, codec.enc.outBuffer = fork, buf
	switch v := obj.(type) {
	case StaticObject:
		v.DefineSSZ(codec)
	case DynamicObject:
		codec.enc.offsetDynamics(v.SizeSSZ(codec.enc.sizer, true))
		v.DefineSSZ(codec)
	default:
		panic(fmt.Sprintf("unsupported type: %T", obj))
	}
	// Retrieve any errors, zero out the sink and return
	err := codec.enc.err

	codec.enc.outBuffer = nil
	codec.enc.err = nil

	return err
}

// DecodeFromStream parses an object with the given size out of a stream. Do not
// use this method with a bytes.Buffer to read from a []byte slice, as that will
// double the byte copying. For that use case, use DecodeFromBytes instead.
func DecodeFromStream(r io.Reader, obj Object, size uint32) error {
	return DecodeFromStreamWithFork(r, obj, size, ForkUnknown)
}

// DecodeFromStreamWithFork is analogous to DecodeFromStream, but allows the user
// to set a specific fork context to decode the object in. This is useful for code-
// bases that have monolith types that unmarshal into many fork formats.
func DecodeFromStreamWithFork(r io.Reader, obj Object, size uint32, fork Fork) error {
	// Retrieve a new decoder codec and set its data source
	codec := decoderPool.Get().(*Codec)
	defer decoderPool.Put(codec)

	codec.fork, codec.dec.inReader = fork, r

	// Start a decoding round with length enforcement in place
	codec.dec.descendIntoSlot(size)

	switch v := obj.(type) {
	case StaticObject:
		v.DefineSSZ(codec)
	case DynamicObject:
		codec.dec.startDynamics(v.SizeSSZ(codec.dec.sizer, true))
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
	return DecodeFromBytesWithFork(blob, obj, ForkUnknown)
}

// DecodeFromBytesWithFork is analogous to DecodeFromBytes, but allows the user
// to set a specific fork context to decode the object in. This is useful for code-
// bases that have monolith types that unmarshal into many fork formats.
func DecodeFromBytesWithFork(blob []byte, obj Object, fork Fork) error {
	// Reject decoding from an empty slice
	if len(blob) == 0 {
		return io.ErrUnexpectedEOF
	}
	// Retrieve a new decoder codec and set its data source
	codec := decoderPool.Get().(*Codec)
	defer decoderPool.Put(codec)

	codec.fork = fork
	codec.dec.inBuffer = blob
	codec.dec.inBufEnd = uintptr(unsafe.Pointer(&blob[0])) + uintptr(len(blob))

	// Start a decoding round with length enforcement in place
	codec.dec.descendIntoSlot(uint32(len(blob)))

	switch v := obj.(type) {
	case StaticObject:
		v.DefineSSZ(codec)
	case DynamicObject:
		codec.dec.startDynamics(v.SizeSSZ(codec.dec.sizer, true))
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
	return HashSequentialWithFork(obj, ForkUnknown)
}

// HashSequentialWithFork is analogous to HashSequential, but allows the user to
// set a specific fork context to hash the object in. This is useful for code-
// bases that have monolith types that hash across many fork formats.
func HashSequentialWithFork(obj Object, fork Fork) [32]byte {
	codec := hasherPool.Get().(*Codec)
	defer hasherPool.Put(codec)
	defer codec.has.Reset()

	codec.fork = fork

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
	return HashConcurrentWithFork(obj, ForkUnknown)
}

// HashConcurrentWithFork is analogous to HashConcurrent, but allows the user to
// set a specific fork context to hash the object in. This is useful for code-
// bases that have monolith types that hash across many fork formats.
func HashConcurrentWithFork(obj Object, fork Fork) [32]byte {
	codec := hasherPool.Get().(*Codec)
	defer hasherPool.Put(codec)
	defer codec.has.Reset()

	codec.fork = fork
	codec.has.threads = true

	codec.has.descendLayer()
	obj.DefineSSZ(codec)
	codec.has.ascendLayer(0)

	if len(codec.has.chunks) != 1 {
		panic(fmt.Sprintf("unfinished hashing: left %v", codec.has.groups))
	}
	codec.has.threads = false
	return codec.has.chunks[0]
}

// Size retrieves the size of a ssz object, independent if it's a static or a
// dynamic one.
func Size(obj Object) uint32 {
	return SizeWithFork(obj, ForkUnknown)
}

// SizeWithFork is analogous to Size, but allows the user to set a specific fork
// context to size the object in. This is useful for codebases that have monolith
// types that serialize across many fork formats.
func SizeWithFork(obj Object, fork Fork) uint32 {
	sizer := sizerPool.Get().(*Sizer)
	defer sizerPool.Put(sizer)

	sizer.codec.fork = fork

	var size uint32
	switch v := obj.(type) {
	case StaticObject:
		size = v.SizeSSZ(sizer)
	case DynamicObject:
		size = v.SizeSSZ(sizer, false)
	default:
		panic(fmt.Sprintf("unsupported type: %T", obj))
	}
	return size
}
