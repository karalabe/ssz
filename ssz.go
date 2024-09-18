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

// EncodeToStream serializes a non-monolithic object into a data stream. If the
// type contains fork-specific rules, use EncodeToStreamOnFork.
//
// Do not use this method with a bytes.Buffer to write into a []byte slice, as
// that will do double the byte copying. For that use case, use EncodeToBytes.
func EncodeToStream(w io.Writer, obj Object) error {
	return EncodeToStreamOnFork(w, obj, ForkUnknown)
}

// EncodeToStreamOnFork serializes a monolithic object into a data stream. If the
// type does not contain fork-specific rules, you can also use EncodeToStream.
//
// Do not use this method with a bytes.Buffer to write into a []byte slice, as that
// will do double the byte copying. For that use case, use EncodeToBytesOnFork.
func EncodeToStreamOnFork(w io.Writer, obj Object, fork Fork) error {
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

// EncodeToBytes serializes a non-monolithic object into a byte buffer. If the
// type contains fork-specific rules, use EncodeToBytesOnFork.
//
// Don't use this method if you want to then write the buffer into a stream via
// some writer, as that would double the memory use for the temporary buffer.
// For that use case, use EncodeToStream.
func EncodeToBytes(buf []byte, obj Object) error {
	return EncodeToBytesOnFork(buf, obj, ForkUnknown)
}

// EncodeToBytesOnFork serializes a monolithic object into a byte buffer. If the
// type does not contain fork-specific rules, you can also use EncodeToBytes.
//
// Don't use this method if you want to then write the buffer into a stream via
// some writer, as that would double the memory use for the temporary buffer.
// For that use case, use EncodeToStreamOnFork.
func EncodeToBytesOnFork(buf []byte, obj Object, fork Fork) error {
	// Sanity check that we have enough space to serialize into
	if size := SizeOnFork(obj, fork); int(size) > len(buf) {
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

// DecodeFromStream parses a non-monolithic object with the given size out of a
// stream. If the type contains fork-specific rules, use DecodeFromStreamOnFork.
//
// Do not use this method with a bytes.Buffer to read from a []byte slice, as that
// will double the byte copying. For that use case, use DecodeFromBytes.
func DecodeFromStream(r io.Reader, obj Object, size uint32) error {
	return DecodeFromStreamOnFork(r, obj, size, ForkUnknown)
}

// DecodeFromStreamOnFork parses a monolithic object with the given size out of
// a stream. If the type does not contain fork-specific rules, you can also use
// DecodeFromStream.
//
// Do not use this method with a bytes.Buffer to read from a []byte slice, as that
// will double the byte copying. For that use case, use DecodeFromBytesOnFork.
func DecodeFromStreamOnFork(r io.Reader, obj Object, size uint32, fork Fork) error {
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

// DecodeFromBytes parses a non-monolithic object from a byte buffer. If the type
// contains fork-specific rules, use DecodeFromBytesOnFork.
//
// Do not use this method if you want to first read the buffer from a stream via
// some reader, as that would double the memory use for the temporary buffer. For
// that use case, use DecodeFromStream instead.
func DecodeFromBytes(blob []byte, obj Object) error {
	return DecodeFromBytesOnFork(blob, obj, ForkUnknown)
}

// DecodeFromBytesOnFork parses a monolithic object from a byte buffer. If the
// type does not contain fork-specific rules, you can also use DecodeFromBytes.
//
// Do not use this method if you want to first read the buffer from a stream via
// some reader, as that would double the memory use for the temporary buffer. For
// that use case, use DecodeFromStreamOnFork instead.
func DecodeFromBytesOnFork(blob []byte, obj Object, fork Fork) error {
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

// HashSequential computes the merkle root of a non-monolithic object on a single
// thread. This is useful for processing small objects with stable runtime and O(1)
// GC guarantees.
//
// If the type contains fork-specific rules, use HashSequentialOnFork.
func HashSequential(obj Object) [32]byte {
	return HashSequentialOnFork(obj, ForkUnknown)
}

// HashSequentialOnFork computes the merkle root of a monolithic object on a single
// thread. This is useful for processing small objects with stable runtime and O(1)
// GC guarantees.
//
// If the type does not contain fork-specific rules, you can also use HashSequential.
func HashSequentialOnFork(obj Object, fork Fork) [32]byte {
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

// HashConcurrent computes the merkle root of a non-monolithic object on potentially
// multiple concurrent threads (iff some data segments are large enough to be worth
// it). This is useful for processing large objects, but will place a bigger load on
// your CPU and GC; and might be more variable timing wise depending on other load.
//
// If the type contains fork-specific rules, use HashConcurrentOnFork.
func HashConcurrent(obj Object) [32]byte {
	return HashConcurrentOnFork(obj, ForkUnknown)
}

// HashConcurrentOnFork computes the merkle root of a monolithic object on potentially
// multiple concurrent threads (iff some data segments are large enough to be worth
// it). This is useful for processing large objects, but will place a bigger load on
// your CPU and GC; and might be more variable timing wise depending on other load.
//
// If the type does not contain fork-specific rules, you can also use HashConcurrent.
func HashConcurrentOnFork(obj Object, fork Fork) [32]byte {
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

// Size retrieves the size of a non-monolithic object, independent if it is static
// or dynamic. If the type contains fork-specific rules, use SizeOnFork.
func Size(obj Object) uint32 {
	return SizeOnFork(obj, ForkUnknown)
}

// SizeOnFork retrieves the size of a monolithic object, independent if it is
// static or dynamic. If the type does not contain fork-specific rules, you can
// also use Size.
func SizeOnFork(obj Object, fork Fork) uint32 {
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
