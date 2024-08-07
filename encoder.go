// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"encoding/binary"
	"io"
	"math/big"
	"unsafe"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
)

// Some helpers to avoid occasional allocations
var (
	boolFalse   = []byte{0x00}
	boolTrue    = []byte{0x01}
	uint256Zero = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

// Encoder is a wrapper around an io.Writer or a []byte buffer to implement SSZ
// encoding in a streaming or buffered way. It has the following behaviors:
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
//     included at the allotted slot.
//
//  4. The contents for dynamic fields are not appended explicitly, rather the
//     caller needs to provide them once more at the end of encoding. This is a
//     design choice to keep the encoder 0-alloc (vs having to stash away the
//     dynamic fields internally).
//
//  5. The encoder does not enforce defined size limits on the dynamic fields.
//     If the caller provided bad data to encode, it is a programming error and
//     a runtime error will not fix anything.
//
// Internally there are a few implementation details that maintainers need to be
// aware of when modifying the code:
//
//  1. The encoder supports two modes of operation: streaming and buffered. Any
//     high level Go code would achieve that with two encoder types implementing
//     a common interface. Unfortunately, the EncodeXYZ methods are using Go's
//     generic system, which is not supported on struct/interface *methods*. As
//     such, `Encoder.EncodeUint64s[T ~uint64](ns []T)` style methods cannot be
//     used, only `EncodeUint64s[T ~uint64](end *Encoder, ns []T)`. The latter
//     form then requires each method internally to do some soft of type cast to
//     handle different encoder implementations. To avoid runtime type asserts,
//     we've opted for a combo encoder with 2 possible outputs and switching on
//     which one is set. Elegant? No. Fast? Yes.
//
//  2. A lot of code snippets are repeated (e.g. encoding the offset, which is
//     the exact same for all the different types, yet the code below has them
//     copied verbatim). Unfortunately the Go compiler doesn't inline functions
//     aggressively enough (neither does it allow explicitly directing it to),
//     and in such tight loops, extra calls matter on performance.
type Encoder[C any] struct {
	outWriter io.Writer // Underlying output stream to write into (streaming mode)
	outBuffer []byte    // Underlying output stream to write into (buffered mode)

	err   error // Any write error to halt future encoding calls
	codec C     // Self-referencing to pass DefineSSZ calls through (API trick)

	buf    [32]byte    // Integer conversion buffer
	bufInt uint256.Int // Big.Int conversion buffer (not pointer, alloc free)

	offset uint32 // Offset tracker for dynamic fields
}

// EncodeBool serializes a boolean.
func EncodeBool[C CodecI[C], T ~bool](enc *Encoder[C], v T) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		if !v {
			_, enc.err = enc.outWriter.Write(boolFalse)
		} else {
			_, enc.err = enc.outWriter.Write(boolTrue)
		}
	} else {
		if !v {
			enc.outBuffer[0] = 0x00
		} else {
			enc.outBuffer[0] = 0x01
		}
		enc.outBuffer = enc.outBuffer[1:]
	}
}

// EncodeUint8 serializes a uint8.
func EncodeUint8[C CodecI[C], T ~uint8](enc *Encoder[C], n T) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		enc.buf[0] = byte(n)
		_, enc.err = enc.outWriter.Write(enc.buf[:1])
	} else {
		enc.outBuffer[0] = byte(n)
		enc.outBuffer = enc.outBuffer[1:]
	}
}

// EncodeUint16 serializes a uint16.
func EncodeUint16[C CodecI[C], T ~uint16](enc *Encoder[C], n T) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint16(enc.buf[:2], (uint16)(n))
		_, enc.err = enc.outWriter.Write(enc.buf[:2])
	} else {
		binary.LittleEndian.PutUint16(enc.outBuffer, (uint16)(n))
		enc.outBuffer = enc.outBuffer[2:]
	}
}

// EncodeUint32 serializes a uint32.
func EncodeUint32[C CodecI[C], T ~uint32](enc *Encoder[C], n T) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint32(enc.buf[:4], (uint32)(n))
		_, enc.err = enc.outWriter.Write(enc.buf[:4])
	} else {
		binary.LittleEndian.PutUint32(enc.outBuffer, (uint32)(n))
		enc.outBuffer = enc.outBuffer[4:]
	}
}

// EncodeUint64 serializes a uint64.
func EncodeUint64[C CodecI[C], T ~uint64](enc *Encoder[C], n T) {
	// Nope, dive into actual encoding
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint64(enc.buf[:8], (uint64)(n))
		_, enc.err = enc.outWriter.Write(enc.buf[:8])
	} else {
		binary.LittleEndian.PutUint64(enc.outBuffer, (uint64)(n))
		enc.outBuffer = enc.outBuffer[8:]
	}
}

// EncodeUint256 serializes a uint256.
//
// Note, a nil pointer is serialized as zero.
func EncodeUint256[C CodecI[C]](enc *Encoder[C], n *uint256.Int) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		if n != nil {
			n.MarshalSSZInto(enc.buf[:32])
			_, enc.err = enc.outWriter.Write(enc.buf[:32])
		} else {
			_, enc.err = enc.outWriter.Write(uint256Zero)
		}
	} else {
		if n != nil {
			n.MarshalSSZInto(enc.outBuffer)
		} else {
			copy(enc.outBuffer, uint256Zero)
		}
		enc.outBuffer = enc.outBuffer[32:]
	}
}

// EncodeUint256BigInt serializes a big.Ing as uint256.
//
// Note, a nil pointer is serialized as zero.
// Note, an overflow will be silently dropped.
func EncodeUint256BigInt[C CodecI[C]](enc *Encoder[C], n *big.Int) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		if n != nil {
			enc.bufInt.SetFromBig(n)
			enc.bufInt.MarshalSSZInto(enc.buf[:32])
			_, enc.err = enc.outWriter.Write(enc.buf[:32])
		} else {
			_, enc.err = enc.outWriter.Write(uint256Zero)
		}
	} else {
		if n != nil {
			enc.bufInt.SetFromBig(n)
			enc.bufInt.MarshalSSZInto(enc.outBuffer)
		} else {
			copy(enc.outBuffer, uint256Zero)
		}
		enc.outBuffer = enc.outBuffer[32:]
	}
}

// EncodeStaticBytes serializes a static binary blob.
//
// The blob is passed by pointer to avoid high stack copy costs and a potential
// escape to the heap.
func EncodeStaticBytes[C CodecI[C], T commonBytesLengths](enc *Encoder[C], blob *T) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		// The code below should have used `*blob[:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		_, enc.err = enc.outWriter.Write(unsafe.Slice(&(*blob)[0], len(*blob)))
	} else {
		// The code below should have used `blob[:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		copy(enc.outBuffer, unsafe.Slice(&(*blob)[0], len(*blob)))
		enc.outBuffer = enc.outBuffer[len(*blob):]
	}
}

// EncodeCheckedStaticBytes serializes a static binary blob.
func EncodeCheckedStaticBytes[C CodecI[C]](enc *Encoder[C], blob []byte) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		_, enc.err = enc.outWriter.Write(blob)
	} else {
		copy(enc.outBuffer, blob)
		enc.outBuffer = enc.outBuffer[len(blob):]
	}
}

// EncodeDynamicBytesOffset serializes a dynamic binary blob.
func EncodeDynamicBytesOffset[C CodecI[C]](enc *Encoder[C], blob []byte) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
		_, enc.err = enc.outWriter.Write(enc.buf[:4])
	} else {
		binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
		enc.outBuffer = enc.outBuffer[4:]
	}
	enc.offset += uint32(len(blob))
}

// EncodeDynamicBytesContent is the lazy data writer for EncodeDynamicBytesOffset.
func EncodeDynamicBytesContent[C CodecI[C]](enc *Encoder[C], blob []byte) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		_, enc.err = enc.outWriter.Write(blob)
	} else {
		copy(enc.outBuffer, blob)
		enc.outBuffer = enc.outBuffer[len(blob):]
	}
}

// EncodeStaticObject serializes a static ssz object.
func EncodeStaticObject[C CodecI[C]](enc *Encoder[C], obj StaticObject[C]) {
	if enc.err != nil {
		return
	}
	obj.DefineSSZ(enc.codec)
}

// EncodeDynamicObjectOffset serializes a dynamic ssz object.
func EncodeDynamicObjectOffset[C CodecI[C]](enc *Encoder[C], obj DynamicObject[C]) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
		_, enc.err = enc.outWriter.Write(enc.buf[:4])
	} else {
		binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
		enc.outBuffer = enc.outBuffer[4:]
	}
	enc.offset += obj.SizeSSZ(false)
}

// EncodeDynamicObjectContent is the lazy data writer for EncodeDynamicObjectOffset.
func EncodeDynamicObjectContent[C CodecI[C]](enc *Encoder[C], obj DynamicObject[C]) {
	if enc.err != nil {
		return
	}
	enc.offsetDynamics(obj.SizeSSZ(true))
	obj.DefineSSZ(enc.codec)
}

// EncodeArrayOfBits serializes a static array of (packed) bits.
func EncodeArrayOfBits[C CodecI[C], T commonBitsLengths](enc *Encoder[C], bits *T) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		// The code below should have used `*bits[:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		_, enc.err = enc.outWriter.Write(unsafe.Slice(&(*bits)[0], len(*bits)))
	} else {
		// The code below should have used `*bits[:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		copy(enc.outBuffer, unsafe.Slice(&(*bits)[0], len(*bits)))
		enc.outBuffer = enc.outBuffer[len(*bits):]
	}
}

// EncodeSliceOfBitsOffset serializes a dynamic slice of (packed) bits.
func EncodeSliceOfBitsOffset[C CodecI[C]](enc *Encoder[C], bits bitfield.Bitlist) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
		_, enc.err = enc.outWriter.Write(enc.buf[:4])
	} else {
		binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
		enc.outBuffer = enc.outBuffer[4:]
	}
	enc.offset += uint32(len(bits))
}

// EncodeSliceOfBitsContent is the lazy data writer for EncodeSliceOfBitsOffset.
func EncodeSliceOfBitsContent[C CodecI[C]](enc *Encoder[C], bits bitfield.Bitlist) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		_, enc.err = enc.outWriter.Write(bits) // bitfield.Bitlist already has the length bit set
	} else {
		copy(enc.outBuffer, bits)
		enc.outBuffer = enc.outBuffer[len(bits):] // bitfield.Bitlist already has the length bit set
	}
}

// EncodeArrayOfUint64s serializes a static array of uint64s.
//
// The reason the ns is passed by pointer and not by value is to prevent it from
// escaping to the heap (and incurring an allocation) when passing it to the
// output stream.
func EncodeArrayOfUint64s[C CodecI[C], T commonUint64sLengths](enc *Encoder[C], ns *T) {
	// The code below should have used `*blob[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	nums := unsafe.Slice(&(*ns)[0], len(*ns))

	// Internally this method is essentially calling EncodeUint64 on all numbers
	// in a loop. Practically, we've inlined that call to make things a *lot* faster.
	if enc.outWriter != nil {
		for _, n := range nums {
			if enc.err != nil {
				return
			}
			binary.LittleEndian.PutUint64(enc.buf[:8], n)
			_, enc.err = enc.outWriter.Write(enc.buf[:8])
		}
	} else {
		for _, n := range nums {
			binary.LittleEndian.PutUint64(enc.outBuffer, n)
			enc.outBuffer = enc.outBuffer[8:]
		}
	}
}

// EncodeSliceOfUint64sOffset serializes a dynamic slice of uint64s.
func EncodeSliceOfUint64sOffset[C CodecI[C], T ~uint64](enc *Encoder[C], ns []T) {
	// Nope, dive into actual encoding
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
		_, enc.err = enc.outWriter.Write(enc.buf[:4])
	} else {
		binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
		enc.outBuffer = enc.outBuffer[4:]
	}
	if items := len(ns); items > 0 {
		enc.offset += uint32(items * 8)
	}
}

// EncodeSliceOfUint64sContent is the lazy data writer for EncodeSliceOfUint64sOffset.
func EncodeSliceOfUint64sContent[C CodecI[C], T ~uint64](enc *Encoder[C], ns []T) {
	if enc.outWriter != nil {
		for _, n := range ns {
			if enc.err != nil {
				return
			}
			binary.LittleEndian.PutUint64(enc.buf[:8], (uint64)(n))
			_, enc.err = enc.outWriter.Write(enc.buf[:8])
		}
	} else {
		for _, n := range ns {
			binary.LittleEndian.PutUint64(enc.outBuffer, (uint64)(n))
			enc.outBuffer = enc.outBuffer[8:]
		}
	}
}

// EncodeArrayOfStaticBytes serializes a static array of static binary
// blobs.
//
// The reason the blobs is passed by pointer and not by value is to prevent it
// from escaping to the heap (and incurring an allocation) when passing it to
// the output stream.
func EncodeArrayOfStaticBytes[T commonBytesArrayLengths[U], U commonBytesLengths, C CodecI[C]](enc *Encoder[C], blobs *T) {
	// The code below should have used `(*blobs)[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	EncodeUnsafeArrayOfStaticBytes(enc, unsafe.Slice(&(*blobs)[0], len(*blobs)))
}

// EncodeUnsafeArrayOfStaticBytes serializes a static array of static binary
// blobs.
func EncodeUnsafeArrayOfStaticBytes[T commonBytesLengths, C CodecI[C]](enc *Encoder[C], blobs []T) {
	// Internally this method is essentially calling EncodeStaticBytes on all
	// the blobs in a loop. Practically, we've inlined that call to make things
	// a *lot* faster.
	if enc.outWriter != nil {
		for i := 0; i < len(blobs); i++ { // don't range loop, T might be an array, copy is expensive
			if enc.err != nil {
				return
			}
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			_, enc.err = enc.outWriter.Write(unsafe.Slice(&blobs[i][0], len(blobs[i])))
		}
	} else {
		for i := 0; i < len(blobs); i++ { // don't range loop, T might be an array, copy is expensive
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			copy(enc.outBuffer, unsafe.Slice(&blobs[i][0], len(blobs[i])))
			enc.outBuffer = enc.outBuffer[len(blobs[i]):]
		}
	}
}

// EncodeCheckedArrayOfStaticBytes serializes a static array of static binary
// blobs.
func EncodeCheckedArrayOfStaticBytes[T commonBytesLengths, C CodecI[C]](enc *Encoder[C], blobs []T) {
	// Internally this method is essentially calling EncodeStaticBytes on all
	// the blobs in a loop. Practically, we've inlined that call to make things
	// a *lot* faster.
	if enc.outWriter != nil {
		for i := 0; i < len(blobs); i++ { // don't range loop, T might be an array, copy is expensive
			if enc.err != nil {
				return
			}
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			_, enc.err = enc.outWriter.Write(unsafe.Slice(&blobs[i][0], len(blobs[i])))
		}
	} else {
		for i := 0; i < len(blobs); i++ { // don't range loop, T might be an array, copy is expensive
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			copy(enc.outBuffer, unsafe.Slice(&blobs[i][0], len(blobs[i])))
			enc.outBuffer = enc.outBuffer[len(blobs[i]):]
		}
	}
}

// EncodeSliceOfStaticBytesOffset serializes a dynamic slice of static binary blobs.
func EncodeSliceOfStaticBytesOffset[C CodecI[C], T commonBytesLengths](enc *Encoder[C], blobs []T) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
		_, enc.err = enc.outWriter.Write(enc.buf[:4])
	} else {
		binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
		enc.outBuffer = enc.outBuffer[4:]
	}
	if items := len(blobs); items > 0 {
		enc.offset += uint32(items * len(blobs[0]))
	}
}

// EncodeSliceOfStaticBytesContent is the lazy data writer for EncodeSliceOfStaticBytesOffset.
func EncodeSliceOfStaticBytesContent[C CodecI[C], T commonBytesLengths](enc *Encoder[C], blobs []T) {
	// Internally this method is essentially calling EncodeStaticBytes on all
	// the blobs in a loop. Practically, we've inlined that call to make things
	// a *lot* faster.
	if enc.outWriter != nil {
		for i := 0; i < len(blobs); i++ { // don't range loop, T might be an array, copy is expensive
			if enc.err != nil {
				return
			}
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			_, enc.err = enc.outWriter.Write(unsafe.Slice(&blobs[i][0], len(blobs[i])))
		}
	} else {
		for i := 0; i < len(blobs); i++ { // don't range loop, T might be an array, copy is expensive
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			copy(enc.outBuffer, unsafe.Slice(&blobs[i][0], len(blobs[i])))
			enc.outBuffer = enc.outBuffer[len(blobs[i]):]
		}
	}
}

// EncodeSliceOfDynamicBytesOffset serializes a dynamic slice of dynamic binary blobs.
func EncodeSliceOfDynamicBytesOffset[C CodecI[C]](enc *Encoder[C], blobs [][]byte) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
		_, enc.err = enc.outWriter.Write(enc.buf[:4])
	} else {
		binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
		enc.outBuffer = enc.outBuffer[4:]
	}
	for _, blob := range blobs {
		enc.offset += uint32(4 + len(blob))
	}
}

// EncodeSliceOfDynamicBytesContent is the lazy data writer for EncodeSliceOfDynamicBytesOffset.
func EncodeSliceOfDynamicBytesContent[C CodecI[C]](enc *Encoder[C], blobs [][]byte) {
	// Nope, dive into actual encoding
	enc.offsetDynamics(uint32(4 * len(blobs)))

	// Inline:
	//
	//	for _, blob := range blobs {
	//		EncodeDynamicBytesOffset(enc, blob)
	//	}
	if enc.outWriter != nil {
		for _, blob := range blobs {
			if enc.err != nil {
				return
			}
			binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
			_, enc.err = enc.outWriter.Write(enc.buf[:4])

			enc.offset += uint32(len(blob))
		}
	} else {
		for _, blob := range blobs {
			binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
			enc.outBuffer = enc.outBuffer[4:]

			enc.offset += uint32(len(blob))
		}
	}
	// Inline:
	//
	// 	for _, blob := range blobs {
	//		EncodeDynamicBytesContent(enc, blob)
	//	}
	if enc.outWriter != nil {
		for _, blob := range blobs {
			if enc.err != nil {
				return
			}
			_, enc.err = enc.outWriter.Write(blob)
		}
	} else {
		for _, blob := range blobs {
			copy(enc.outBuffer, blob)
			enc.outBuffer = enc.outBuffer[len(blob):]
		}
	}
}

// EncodeSliceOfStaticObjectsOffset serializes a dynamic slice of static ssz objects.
func EncodeSliceOfStaticObjectsOffset[C CodecI[C], T StaticObject[C]](enc *Encoder[C], objects []T) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
		_, enc.err = enc.outWriter.Write(enc.buf[:4])
	} else {
		binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
		enc.outBuffer = enc.outBuffer[4:]
	}
	if items := len(objects); items > 0 {
		enc.offset += uint32(items) * objects[0].SizeSSZ()
	}
}

// EncodeSliceOfStaticObjectsContent is the lazy data writer for EncodeSliceOfStaticObjectsOffset.
func EncodeSliceOfStaticObjectsContent[C CodecI[C], T StaticObject[C]](enc *Encoder[C], objects []T) {
	for _, obj := range objects {
		if enc.err != nil {
			return
		}
		obj.DefineSSZ(enc.codec)
	}
}

// EncodeSliceOfDynamicObjectsOffset serializes a dynamic slice of dynamic ssz objects.
func EncodeSliceOfDynamicObjectsOffset[C CodecI[C], T DynamicObject[C]](enc *Encoder[C], objects []T) {
	if enc.outWriter != nil {
		if enc.err != nil {
			return
		}
		binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
		_, enc.err = enc.outWriter.Write(enc.buf[:4])
	} else {
		binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
		enc.outBuffer = enc.outBuffer[4:]
	}
	for _, obj := range objects {
		enc.offset += 4 + obj.SizeSSZ(false)
	}
}

// EncodeSliceOfDynamicObjectsContent is the lazy data writer for EncodeSliceOfDynamicObjectsOffset.
func EncodeSliceOfDynamicObjectsContent[C CodecI[C], T DynamicObject[C]](enc *Encoder[C], objects []T) {
	enc.offsetDynamics(uint32(4 * len(objects)))

	// Inline:
	//
	// 	for _, obj := range objects {
	//		EncodeDynamicObjectOffset(enc, obj)
	//	}
	if enc.outWriter != nil {
		for _, obj := range objects {
			if enc.err != nil {
				return
			}
			binary.LittleEndian.PutUint32(enc.buf[:4], enc.offset)
			_, enc.err = enc.outWriter.Write(enc.buf[:4])

			enc.offset += obj.SizeSSZ(false)
		}
	} else {
		for _, obj := range objects {
			binary.LittleEndian.PutUint32(enc.outBuffer, enc.offset)
			enc.outBuffer = enc.outBuffer[4:]

			enc.offset += obj.SizeSSZ(false)
		}
	}
	// Inline:
	//
	// 	for _, obj := range objects {
	//		EncodeDynamicObjectContent(enc, obj)
	//	}
	for _, obj := range objects {
		if enc.err != nil {
			return
		}
		enc.offsetDynamics(obj.SizeSSZ(true))
		obj.DefineSSZ(enc.codec)
	}
}

// offsetDynamics marks the item being encoded as a dynamic type, setting the starting
// offset for the dynamic fields.
func (enc *Encoder[C]) offsetDynamics(offset uint32) {
	enc.offset = offset
}
