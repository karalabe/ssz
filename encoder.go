// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"encoding/binary"
	"io"
	"unsafe"

	"github.com/holiman/uint256"
)

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
//     included at the allotted slot. The writes themselves will be done later.
//
//  4. The encoder does not enforce defined size limits on the dynamic fields.
//     If the caller provided bad data to encode, it is a programming error and
//     a runtime error will not fix anything.
type Encoder struct {
	out io.Writer // Underlying output stream to write into
	err error     // Any write error to halt future encoding calls

	codec *Codec   // Self-referencing to pass DefineSSZ calls through (API trick)
	buf   [32]byte // Integer conversion buffer

	offset uint32     // Offset tracker for dynamic fields
	pend   []func()   // Queue of dynamics pending to be encoded
	pends  [][]func() // Stack of dynamics queues from outer calls
}

// EncodeUint64 serializes a uint64.
func EncodeUint64[T ~uint64](enc *Encoder, n T) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint64(enc.buf[:8], (uint64)(n))
	_, enc.err = enc.out.Write(enc.buf[:8])
}

// EncodeUint256 serializes a uint256.
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

// EncodeStaticBytes serializes a static binary blob.
func EncodeStaticBytes(enc *Encoder, blob []byte) {
	if enc.err != nil {
		return
	}
	_, enc.err = enc.out.Write(blob)
}

// EncodeDynamicBytes serializes a dynamic binary blob.
func EncodeDynamicBytes(enc *Encoder, blob []byte) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])
	enc.offset += uint32(len(blob))

	enc.pend = append(enc.pend, func() { EncodeStaticBytes(enc, blob) })
}

// EncodeStaticObject serializes a static ssz object.
func EncodeStaticObject(enc *Encoder, obj StaticObject) {
	if enc.err != nil {
		return
	}
	obj.DefineSSZ(enc.codec)
}

// EncodeDynamicObject serializes a dynamic ssz object.
func EncodeDynamicObject(enc *Encoder, obj DynamicObject) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])
	enc.offset += obj.SizeSSZ(false)

	enc.pend = append(enc.pend, func() { encodeDynamicObject(enc, obj) })
}

// encodeDynamicObject is the lazy data writer for EncodeDynamicObject.
func encodeDynamicObject(enc *Encoder, obj DynamicObject) {
	if enc.err != nil {
		return
	}
	enc.startDynamics(obj.SizeSSZ(true))
	obj.DefineSSZ(enc.codec)
	enc.flushDynamics()
}

// EncodeSliceOfUint64s serializes a dynamic slice of uint64s.
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

// encodeSliceOfUint64s is the lazy data writer for EncodeSliceOfUint64s.
func encodeSliceOfUint64s[T ~uint64](enc *Encoder, ns []T) {
	if enc.err != nil {
		return
	}
	for _, n := range ns {
		EncodeUint64(enc, n)
	}
}

// EncodeArrayOfStaticBytes serializes a static array of static binary blobs.
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

// EncodeSliceOfStaticBytes serializes a dynamic slice of static binary blobs.
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

// encodeSliceOfStaticBytes is the lazy data writer for EncodeSliceOfStaticBytes.
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

// EncodeSliceOfDynamicBytes serializes a dynamic slice of dynamic binary blobs.
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

// encodeSliceOfDynamicBytes is the lazy data writer for EncodeSliceOfDynamicBytes.
func encodeSliceOfDynamicBytes(enc *Encoder, blobs [][]byte) {
	if enc.err != nil {
		return
	}
	enc.startDynamics(uint32(4 * len(blobs)))
	for _, blob := range blobs {
		EncodeDynamicBytes(enc, blob)
	}
	enc.flushDynamics()
}

// EncodeSliceOfStaticObjects serializes a dynamic slice of static ssz objects.
func EncodeSliceOfStaticObjects[T StaticObject](enc *Encoder, objects []T) {
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

// encodeSliceOfStaticObjects is the lazy data writer for EncodeSliceOfStaticObjects.
func encodeSliceOfStaticObjects[T StaticObject](enc *Encoder, objects []T) {
	if enc.err != nil {
		return
	}
	for _, obj := range objects {
		obj.DefineSSZ(enc.codec)
	}
}

// EncodeSliceOfDynamicObjects serializes a dynamic slice of dynamic ssz objects.
func EncodeSliceOfDynamicObjects[T DynamicObject](enc *Encoder, objects []T) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])

	for _, obj := range objects {
		enc.offset += 4 + obj.SizeSSZ(false)
	}
	enc.pend = append(enc.pend, func() { encodeSliceOfDynamicObjects(enc, objects) })
}

// encodeSliceOfDynamicObjects is the lazy data writer for EncodeSliceOfDynamicObjects.
func encodeSliceOfDynamicObjects[T DynamicObject](enc *Encoder, objects []T) {
	if enc.err != nil {
		return
	}
	enc.startDynamics(uint32(4 * len(objects)))
	for _, obj := range objects {
		EncodeDynamicObject(enc, obj)
	}
	enc.flushDynamics()
}

// startDynamics marks the item being encoded as a dynamic type, setting the starting
// offset for the dynamic fields.
func (enc *Encoder) startDynamics(offset uint32) {
	enc.offset = offset

	// Try to reuse older pending slices to avoid allocations
	n := len(enc.pends)

	if cap(enc.pends) > n {
		enc.pends = enc.pends[:n+1]
		enc.pend, enc.pends[n] = enc.pends[n], enc.pend
	} else {
		enc.pends = append(enc.pends, enc.pend)
		enc.pend = nil
	}
}

// flushDynamics marks the end of the dynamic fields, encoding anything queued up and
// restoring any previous states for outer call continuation.
func (enc *Encoder) flushDynamics() {
	// Apply any delayed ops and clear them out
	for _, pend := range enc.pend {
		pend()
	}
	enc.pend = enc.pend[:0]

	// Restore the previous pends, but swap in the current slice as a future memcache
	last := len(enc.pends) - 1

	enc.pend, enc.pends[last] = enc.pends[last], enc.pend
	enc.pends = enc.pends[:last]
}
