// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"encoding/binary"
	"io"
	"sync"
	"unsafe"

	"github.com/holiman/uint256"
)

// subEncoderPool is a pool of blank SSZ codecs to use when an encoder needs to
// descend into an object via a codec interface.
//
// Note, this is different from encoderPool, do not mix and match!
var subEncoderPool = sync.Pool{
	New: func() any { return new(Codec) },
}

// Encoder is a wrapper around an io.Writer to implement dense SSZ encoding. It
// has the following behaviors:
//
//  1. The encoder does not buffer, simply writes to the wrapped output stream
//     directly. If you need buffering (and flushing), that is up to you.
//
//  2. The encoder does not return errors that were hit during writing to the
//     underlying output stream from individual encoding methods. Since there
//     is no expectation (in general) for failure, user code can be denser if
//     error checking is done at the end. Internally, of course, an error will
//     halt all future output operations.
//
//  3. The offsets for dynamic fields are tracked internally by the encoder, so
//     the caller only needs to provide the field, the offset of which should be
//     included at the alloted slot. The writes themselves should be done later.
//
//  4. The encoder does not enforce defined size limits on the dynamic fields.
//     If the caller provided bad data to encode, it is a programming error and
//     a runtime error will not fix anything.
type Encoder struct {
	out io.Writer // Underlying output stream to write into
	err error     // Any write error to halt future encoding calls
	dyn bool      // Whether dynamics were found encoding
	buf [32]byte  // Integer conversion buffer

	offset  uint32     // Offset tracker for dynamic fields
	offsets []uint32   // Stack of offsets from outer calls
	pend    []func()   // Queue of dynamics pending to be encoded
	pends   [][]func() // Stack of dynamics queues from outer calls
}

// OffsetDynamics marks the item being encoded as a dynamic type, setting the starting
// offset for the dynamic fields.
func (enc *Encoder) OffsetDynamics(offset int) func() {
	enc.dyn = true

	enc.offsets = append(enc.offsets, enc.offset)
	enc.offset = uint32(offset)
	enc.pends = append(enc.pends, enc.pend)
	enc.pend = nil

	return enc.dynamicDone
}

// dynamicDone marks the end of the dyamic fields, encoding anything queued up and
// restoring any previous states for outer call continuation.
func (enc *Encoder) dynamicDone() {
	for _, pend := range enc.pend {
		pend()
	}
	enc.pend = enc.pends[len(enc.pends)-1]
	enc.pends = enc.pends[:len(enc.pends)-1]
	enc.offset = enc.offsets[len(enc.offsets)-1]
	enc.offsets = enc.offsets[:len(enc.offsets)-1]
}

// EncodeUint64 serializes a uint64 as little-endian.
func EncodeUint64[T ~uint64](enc *Encoder, n T) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint64(enc.buf[:8], (uint64)(n))
	_, enc.err = enc.out.Write(enc.buf[:8])
}

// EncodeUint256 serializes a uint256 as little-endian.
func EncodeUint256(enc *Encoder, n *uint256.Int) {
	if enc.err != nil {
		return
	}
	// There might be degenerate cases where n was not initialized. Whilst we
	// *could* panic, it's probably cleaner to assume it's zero.
	if n != nil {
		n.MarshalSSZTo(enc.buf[:32])
	} else {
		copy(enc.buf[:], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	}
	_, enc.err = enc.out.Write(enc.buf[:32])
}

// EncodeStaticBytes serializes raw bytes as is.
func EncodeStaticBytes(enc *Encoder, blob []byte) {
	if enc.err != nil {
		return
	}
	_, enc.err = enc.out.Write(blob)
}

// EncodeDynamicBytes serializes the current offset as a uint32 little-endian,
// and shifts it by the size of the blob.
//
// Later when all the static fields have been written out, the dynamic content
// will also be flushed. Make sure you called Encoder.OffsetDynamics and defer-ed the
// return lambda.
func EncodeDynamicBytes(enc *Encoder, blob []byte) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])
	enc.offset += uint32(len(blob))

	enc.pend = append(enc.pend, func() { EncodeStaticBytes(enc, blob) })
}

// EncodeStaticObject serializes the given static object into the ssz stream.
func EncodeStaticObject(enc *Encoder, obj Object) {
	if enc.err != nil {
		return
	}
	codec := subEncoderPool.Get().(*Codec)
	defer subEncoderPool.Put(codec)

	codec.enc = enc
	obj.DefineSSZ(codec)
}

// EncodeDynamicObject serializes the given dynamic object into the ssz stream.
func EncodeDynamicObject(enc *Encoder, obj Object) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])
	enc.offset += obj.SizeSSZ()

	enc.pend = append(enc.pend, func() { encodeDynamicObject(enc, obj) })
}

// encodeDynamicObject serializes the given dynamic object into the ssz stream.
func encodeDynamicObject(enc *Encoder, obj Object) {
	if enc.err != nil {
		return
	}
	codec := subEncoderPool.Get().(*Codec)
	defer subEncoderPool.Put(codec)

	codec.enc = enc
	obj.DefineSSZ(codec)
}

// EncodeSliceOfUint64s serializes a dynamic slice of uint64s into the ssz stream
func EncodeSliceOfUint64s[T ~uint64](enc *Encoder, ns []T) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])

	if items := len(ns); items > 0 {
		enc.offset += uint32(items * 8)
	}
	enc.pend = append(enc.pend, func() { encodeSliceOfUint64s(enc, ns) })
}

// encodeSliceOfUint64s serializes a slice of static objects by simply iterating
// the slice and serializing each individually.
func encodeSliceOfUint64s[T ~uint64](enc *Encoder, ns []T) {
	if enc.err != nil {
		return
	}
	for _, n := range ns {
		EncodeUint64(enc, n)
	}
}

// EncodeArrayOfStaticBytes serializes a static number of static bytes.
func EncodeArrayOfStaticBytes[T commonBinaryLengths](enc *Encoder, blobs []T) {
	if enc.err != nil {
		return
	}
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		EncodeStaticBytes(enc, unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
}

// EncodeSliceOfStaticBytes serializes the current offset as a uint32 little-endian,
// and shifts if by the cumulative length of the static binary slices needed to
// encode them.
func EncodeSliceOfStaticBytes[T commonBinaryLengths](enc *Encoder, blobs []T) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])

	if items := len(blobs); items > 0 {
		enc.offset += uint32(items * len(blobs[0]))
	}
	enc.pend = append(enc.pend, func() { encodeSliceOfStaticBytes(enc, blobs) })
}

// encodeSliceOfStaticBytes serializes a slice of static objects by simply iterating
// the slice and serializing each individually.
func encodeSliceOfStaticBytes[T commonBinaryLengths](enc *Encoder, blobs []T) {
	if enc.err != nil {
		return
	}
	for _, blob := range blobs {
		// The code below should have used `blob[:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		EncodeStaticBytes(enc, unsafe.Slice(&blob[0], len(blob)))
	}
}

// EncodeSliceOfDynamicBytes serializes the current offset as a uint32 little-endian, and
// shifts if by the cumulative length of the binary slices and the offsets
// needed to encode them.
//
// Later when all the static fields have been written out, the dynamic content
// will also be flushed. Make sure you called Encoder.OffsetDynamics and defer-ed the
// return lambda.
func EncodeSliceOfDynamicBytes(enc *Encoder, blobs [][]byte) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])
	for _, blob := range blobs {
		enc.offset += uint32(4 + len(blob))
	}
	enc.pend = append(enc.pend, func() { encodeSliceOfDynamicBytes(enc, blobs) })
}

// encodeSliceOfDynamicBytes serializes a slice of dynamic blobs by first writing all
// the individual offsets, and then writing the dynamic data itself.
func encodeSliceOfDynamicBytes(enc *Encoder, blobs [][]byte) {
	if enc.err != nil {
		return
	}
	defer enc.OffsetDynamics(4 * len(blobs))()

	for _, blob := range blobs {
		EncodeDynamicBytes(enc, blob)
	}
}

// EncodeSliceOfStaticObjects serializes the current offset as a uint32 little-endian, and
// shifts if by the cumulative length of the fixed size objects.
//
// Later when all the static fields have been written out, the dynamic content
// will also be flushed. Make sure you called Encoder.OffsetDynamics and defer-ed the
// return lambda.
func EncodeSliceOfStaticObjects[T newableObject[U], U any](enc *Encoder, objects []T) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])

	if items := len(objects); items > 0 {
		enc.offset += uint32(items) * objects[0].SizeSSZ()
	}
	enc.pend = append(enc.pend, func() { encodeSliceOfStaticObjects(enc, objects) })
}

// encodeSliceOfStaticObjects serializes a slice of static objects by simply iterating
// the slice and serializing each individually.
func encodeSliceOfStaticObjects[T Object](enc *Encoder, objects []T) {
	if enc.err != nil {
		return
	}
	codec := subEncoderPool.Get().(*Codec)
	defer subEncoderPool.Put(codec)

	codec.enc = enc
	for _, obj := range objects {
		obj.DefineSSZ(codec)
	}
}
