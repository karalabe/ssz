// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"encoding/binary"
	"io"
	"math/big"

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
//     included at the alloted slot. The writes themselves should be done later.
//
//  4. The encoder does not enforce defined size limits on the dynamic fields.
//     If the caller provided bad data to encode, it is a programming error and
//     a runtime error will not fix anything.
type Encoder struct {
	out io.Writer // Underlying output stream to write into
	err error     // Any write error to halt future encoding calls

	buf  [32]byte    // Integer conversion buffer
	ibuf uint256.Int // Big integer conversion buffer

	offset  uint32     // Offset tracker for dynamic fields
	offsets []uint32   // Stack of offsets from outer calls
	pend    []func()   // Queue of dynamics pending to be encoded
	pends   [][]func() // Stack of dynamics queues from outer calls
}

// OffsetDynamics marks the item being encoded as a dynamic type, setting the starting
// offset for the dynamic fields.
func (enc *Encoder) OffsetDynamics(offset int) func() {
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
func EncodeUint64(enc *Encoder, n uint64) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint64(enc.buf[:8], n)
	_, enc.err = enc.out.Write(enc.buf[:8])
}

// EncodeBigInt serializes a uint256 as little-endian.
func EncodeBigInt(enc *Encoder, n *big.Int) {
	if enc.err != nil {
		return
	}
	enc.ibuf.SetFromBig(n)
	enc.ibuf.MarshalSSZTo(enc.buf[:32])
	_, enc.err = enc.out.Write(enc.buf[:32])
}

// EncodeBinary serializes raw bytes as is.
func EncodeBinary(enc *Encoder, bytes []byte) {
	if enc.err != nil {
		return
	}
	_, enc.err = enc.out.Write(bytes)
}

// EncodeDynamicBlob serializes the current offset as a uint32 little-endian,
// and shifts it by the size of the blob.
//
// Later when all the static fields have been written out, the dynamic content
// will also be flushed. Make sure you called Encoder.OffsetDynamics and defer-ed the
// return lambda.
func EncodeDynamicBlob(enc *Encoder, blob []byte) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])
	enc.offset += uint32(len(blob))

	enc.pend = append(enc.pend, func() { EncodeBinary(enc, blob) })
}

// EncodeDynamicBlobs serializes the current offset as a uint32 little-endian, and
// shifts if by the cummulative length of the binary slices and the offsets
// needed to encode them.
//
// Later when all the static fields have been written out, the dynamic content
// will also be flushed. Make sure you called Encoder.OffsetDynamics and defer-ed the
// return lambda.
func EncodeDynamicBlobs(enc *Encoder, blobs [][]byte) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])
	for _, blob := range blobs {
		enc.offset += uint32(4 + len(blob))
	}
	enc.pend = append(enc.pend, func() { encodeDynamicBlobs(enc, blobs) })
}

// encodeDynamicBlobs serializes a slice of dynamic blobs by first writing all
// the individual offsets, and then writing the dynamic data itself.
func encodeDynamicBlobs(enc *Encoder, blobs [][]byte) {
	if enc.err != nil {
		return
	}
	defer enc.OffsetDynamics(4 * len(blobs))()

	for _, blob := range blobs {
		EncodeDynamicBlob(enc, blob)
	}
}

// EncodeDynamicStatics serializes the current offset as a uint32 little-endian, and
// shifts if by the cummulative length of the fixed size objects.
//
// Later when all the static fields have been written out, the dynamic content
// will also be flushed. Make sure you called Encoder.OffsetDynamics and defer-ed the
// return lambda.
func EncodeDynamicStatics[T Object](enc *Encoder, objects []T) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
	_, enc.err = enc.out.Write(enc.buf[:4])

	if items := len(objects); items > 0 {
		enc.offset += uint32(items) * objects[0].SizeSSZ()
	}
	enc.pend = append(enc.pend, func() { encodeDynamicStatics(enc, objects) })
}

// encodeDynamicStatics serializes a slice of static iojects by simply iterating
// the slice and serializing each individually.
func encodeDynamicStatics[T Object](enc *Encoder, objects []T) {
	if enc.err != nil {
		return
	}
	for _, obj := range objects {
		obj.EncodeSSZ(enc)
	}
}
