// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"math/big"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
)

type CodecI interface {
	Enc() *Encoder
	Dec() *Decoder
	Has() *Hasher
	DefineEncoder(impl func(enc *Encoder))
	DefineDecoder(impl func(dec *Decoder))
	DefineHasher(impl func(has *Hasher))
}

// Codec is a unified SSZ encoder and decoder that allows simple structs to
// define their schemas once and have that work for both operations at once
// (with the same speed as explicitly typing them out would, of course).
type Codec struct {
	enc *Encoder
	dec *Decoder
	has *Hasher
}

func (c *Codec) Enc() *Encoder {
	return c.enc
}

func (c *Codec) Dec() *Decoder {
	return c.dec
}

func (c *Codec) Has() *Hasher {
	return c.has
}

// DefineEncoder uses a dedicated encoder in case the types SSZ conversion is for
// some reason asymmetric (e.g. encoding depends on fields, decoding depends on
// outer context).
//
// In reality, it will be the live code run when the object is being serialized.
func (c *Codec) DefineEncoder(impl func(enc *Encoder)) {
	if c.enc != nil {
		impl(c.enc)
	}
}

// DefineDecoder uses a dedicated decoder in case the types SSZ conversion is for
// some reason asymmetric (e.g. encoding depends on fields, decoding depends on
// outer context).
//
// In reality, it will be the live code run when the object is being parsed.
func (c *Codec) DefineDecoder(impl func(dec *Decoder)) {
	if c.dec != nil {
		impl(c.dec)
	}
}

// DefineHasher uses a dedicated hasher in case the types SSZ conversion is for
// some reason asymmetric (e.g. encoding depends on fields, decoding depends on
// outer context).
//
// In reality, it will be the live code run when the object is being parsed.
func (c *Codec) DefineHasher(impl func(has *Hasher)) {
	if c.has != nil {
		impl(c.has)
	}
}

// DefineBool defines the next field as a 1 byte boolean.
func DefineBool[T ~bool](c CodecI, v *T) {
	if c.Enc() != nil {
		EncodeBool(c.Enc(), *v)
		return
	}
	if c.Dec() != nil {
		DecodeBool(c.Dec(), v)
		return
	}
	HashBool(c.Has(), *v)
}

// DefineUint8 defines the next field as a uint8.
func DefineUint8[T ~uint8](c CodecI, n *T) {
	if c.Enc() != nil {
		EncodeUint8(c.Enc(), *n)
		return
	}
	if c.Dec() != nil {
		DecodeUint8(c.Dec(), n)
		return
	}
	HashUint8(c.Has(), *n)
}

// DefineUint16 defines the next field as a uint16.
func DefineUint16[T ~uint16](c CodecI, n *T) {
	if c.Enc() != nil {
		EncodeUint16(c.Enc(), *n)
		return
	}
	if c.Dec() != nil {
		DecodeUint16(c.Dec(), n)
		return
	}
	HashUint16(c.Has(), *n)
}

// DefineUint32 defines the next field as a uint32.
func DefineUint32[T ~uint32](c CodecI, n *T) {
	if c.Enc() != nil {
		EncodeUint32(c.Enc(), *n)
		return
	}
	if c.Dec() != nil {
		DecodeUint32(c.Dec(), n)
		return
	}
	HashUint32(c.Has(), *n)
}

// DefineUint64 defines the next field as a uint64.
func DefineUint64[T ~uint64](c CodecI, n *T) {
	if c.Enc() != nil {
		EncodeUint64(c.Enc(), *n)
		return
	}
	if c.Dec() != nil {
		DecodeUint64(c.Dec(), n)
		return
	}
	HashUint64(c.Has(), *n)
}

// DefineUint256 defines the next field as a uint256.
func DefineUint256(c CodecI, n **uint256.Int) {
	if c.Enc() != nil {
		EncodeUint256(c.Enc(), *n)
		return
	}
	if c.Dec() != nil {
		DecodeUint256(c.Dec(), n)
		return
	}
	HashUint256(c.Has(), *n)
}

// DefineUint256BigInt defines the next field as a uint256.
func DefineUint256BigInt(c CodecI, n **big.Int) {
	if c.Enc() != nil {
		EncodeUint256BigInt(c.Enc(), *n)
		return
	}
	if c.Dec() != nil {
		DecodeUint256BigInt(c.Dec(), n)
		return
	}
	HashUint256BigInt(c.Has(), *n)
}

// DefineStaticBytes defines the next field as static binary blob. This method
// can be used for byte arrays.
func DefineStaticBytes[T commonBytesLengths](c CodecI, blob *T) {
	if c.Enc() != nil {
		EncodeStaticBytes(c.Enc(), blob)
		return
	}
	if c.Dec() != nil {
		DecodeStaticBytes(c.Dec(), blob)
		return
	}
	HashStaticBytes(c.Has(), blob)
}

// DefineCheckedStaticBytes defines the next field as static binary blob. This
// method can be used for plain byte slices, which is more expensive, since it
// needs runtime size validation.
func DefineCheckedStaticBytes(c CodecI, blob *[]byte, size uint64) {
	if c.Enc() != nil {
		EncodeCheckedStaticBytes(c.Enc(), *blob)
		return
	}
	if c.Dec() != nil {
		DecodeCheckedStaticBytes(c.Dec(), blob, size)
		return
	}
	HashCheckedStaticBytes(c.Has(), *blob)
}

// DefineDynamicBytesOffset defines the next field as dynamic binary blob.
func DefineDynamicBytesOffset(c CodecI, blob *[]byte, maxSize uint64) {
	if c.Enc() != nil {
		EncodeDynamicBytesOffset(c.Enc(), *blob)
		return
	}
	if c.Dec() != nil {
		DecodeDynamicBytesOffset(c.Dec(), blob)
		return
	}
	HashDynamicBytes(c.Has(), *blob, maxSize)
}

// DefineDynamicBytesContent defines the next field as dynamic binary blob.
func DefineDynamicBytesContent(c CodecI, blob *[]byte, maxSize uint64) {
	if c.Enc() != nil {
		EncodeDynamicBytesContent(c.Enc(), *blob)
		return
	}
	if c.Dec() != nil {
		DecodeDynamicBytesContent(c.Dec(), blob, maxSize)
		return
	}
	// No hashing, done at the offset position
}

// DefineStaticObject defines the next field as a static ssz object.
func DefineStaticObject[T newableStaticObject[U], U any](c CodecI, obj *T) {
	if c.Enc() != nil {
		EncodeStaticObject(c.Enc(), *obj)
		return
	}
	if c.Dec() != nil {
		DecodeStaticObject(c.Dec(), obj)
		return
	}
	HashStaticObject(c.Has(), *obj)
}

// DefineDynamicObjectOffset defines the next field as a dynamic ssz object.
func DefineDynamicObjectOffset[T newableDynamicObject[U], U any](c CodecI, obj *T) {
	if c.Enc() != nil {
		EncodeDynamicObjectOffset(c.Enc(), *obj)
		return
	}
	if c.Dec() != nil {
		DecodeDynamicObjectOffset(c.Dec(), obj)
		return
	}
	HashDynamicObject(c.Has(), *obj)
}

// DefineDynamicObjectContent defines the next field as a dynamic ssz object.
func DefineDynamicObjectContent[T newableDynamicObject[U], U any](c CodecI, obj *T) {
	if c.Enc() != nil {
		EncodeDynamicObjectContent(c.Enc(), *obj)
		return
	}
	if c.Dec() != nil {
		DecodeDynamicObjectContent(c.Dec(), obj)
		return
	}
	// No hashing, done at the offset position
}

// DefineArrayOfBits defines the next field as a static array of (packed) bits.
func DefineArrayOfBits[T commonBitsLengths](c CodecI, bits *T, size uint64) {
	if c.Enc() != nil {
		EncodeArrayOfBits(c.Enc(), bits)
		return
	}
	if c.Dec() != nil {
		DecodeArrayOfBits(c.Dec(), bits, size)
		return
	}
	HashArrayOfBits(c.Has(), bits)
}

// DefineSliceOfBitsOffset defines the next field as a dynamic slice of (packed) bits.
func DefineSliceOfBitsOffset(c CodecI, bits *bitfield.Bitlist, maxBits uint64) {
	if c.Enc() != nil {
		EncodeSliceOfBitsOffset(c.Enc(), *bits)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfBitsOffset(c.Dec(), bits)
		return
	}
	HashSliceOfBits(c.Has(), *bits, maxBits)
}

// DefineSliceOfBitsContent defines the next field as a dynamic slice of (packed) bits.
func DefineSliceOfBitsContent(c CodecI, bits *bitfield.Bitlist, maxBits uint64) {
	if c.Enc() != nil {
		EncodeSliceOfBitsContent(c.Enc(), *bits)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfBitsContent(c.Dec(), bits, maxBits)
		return
	}
	// No hashing, done at the offset position
}

// DefineArrayOfUint64s defines the next field as a static array of uint64s.
func DefineArrayOfUint64s[T commonUint64sLengths](c CodecI, ns *T) {
	if c.Enc() != nil {
		EncodeArrayOfUint64s(c.Enc(), ns)
		return
	}
	if c.Dec() != nil {
		DecodeArrayOfUint64s(c.Dec(), ns)
		return
	}
	HashArrayOfUint64s(c.Has(), ns)
}

// DefineSliceOfUint64sOffset defines the next field as a dynamic slice of uint64s.
func DefineSliceOfUint64sOffset[T ~uint64](c CodecI, ns *[]T, maxItems uint64) {
	if c.Enc() != nil {
		EncodeSliceOfUint64sOffset(c.Enc(), *ns)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfUint64sOffset(c.Dec(), ns)
		return
	}
	HashSliceOfUint64s(c.Has(), *ns, maxItems)
}

// DefineSliceOfUint64sContent defines the next field as a dynamic slice of uint64s.
func DefineSliceOfUint64sContent[T ~uint64](c CodecI, ns *[]T, maxItems uint64) {
	if c.Enc() != nil {
		EncodeSliceOfUint64sContent(c.Enc(), *ns)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfUint64sContent(c.Dec(), ns, maxItems)
		return
	}
	// No hashing, done at the offset position
}

// DefineArrayOfStaticBytes defines the next field as a static array of static
// binary blobs.
func DefineArrayOfStaticBytes[T commonBytesArrayLengths[U], U commonBytesLengths](c CodecI, blobs *T) {
	if c.Enc() != nil {
		EncodeArrayOfStaticBytes[T, U](c.Enc(), blobs)
		return
	}
	if c.Dec() != nil {
		DecodeArrayOfStaticBytes[T, U](c.Dec(), blobs)
		return
	}
	HashArrayOfStaticBytes[T, U](c.Has(), blobs)
}

// DefineUnsafeArrayOfStaticBytes defines the next field as a static array of
// static binary blobs. This method operates on plain slices of byte arrays and
// will crash if provided a slice of a non-array. Its purpose is to get around
// Go's generics limitations in generated code (use DefineArrayOfStaticBytes).
func DefineUnsafeArrayOfStaticBytes[T commonBytesLengths](c CodecI, blobs []T) {
	if c.Enc() != nil {
		EncodeUnsafeArrayOfStaticBytes(c.Enc(), blobs)
		return
	}
	if c.Dec() != nil {
		DecodeUnsafeArrayOfStaticBytes(c.Dec(), blobs)
		return
	}
	HashUnsafeArrayOfStaticBytes(c.Has(), blobs)
}

// DefineCheckedArrayOfStaticBytes defines the next field as a static array of
// static binary blobs. This method can be used for plain slices of byte arrays,
// which is more expensive since it needs runtime size validation.
func DefineCheckedArrayOfStaticBytes[T commonBytesLengths](c CodecI, blobs *[]T, size uint64) {
	if c.Enc() != nil {
		EncodeCheckedArrayOfStaticBytes(c.Enc(), *blobs)
		return
	}
	if c.Dec() != nil {
		DecodeCheckedArrayOfStaticBytes(c.Dec(), blobs, size)
		return
	}
	HashCheckedArrayOfStaticBytes(c.Has(), *blobs)
}

// DefineSliceOfStaticBytesOffset defines the next field as a dynamic slice of static
// binary blobs.
func DefineSliceOfStaticBytesOffset[T commonBytesLengths](c CodecI, bytes *[]T, maxItems uint64) {
	if c.Enc() != nil {
		EncodeSliceOfStaticBytesOffset(c.Enc(), *bytes)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfStaticBytesOffset(c.Dec(), bytes)
		return
	}
	HashSliceOfStaticBytes(c.Has(), *bytes, maxItems)
}

// DefineSliceOfStaticBytesContent defines the next field as a dynamic slice of static
// binary blobs.
func DefineSliceOfStaticBytesContent[T commonBytesLengths](c CodecI, blobs *[]T, maxItems uint64) {
	if c.Enc() != nil {
		EncodeSliceOfStaticBytesContent(c.Enc(), *blobs)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfStaticBytesContent(c.Dec(), blobs, maxItems)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfDynamicBytesOffset defines the next field as a dynamic slice of dynamic
// binary blobs.
func DefineSliceOfDynamicBytesOffset(c CodecI, blobs *[][]byte, maxItems uint64, maxSize uint64) {
	if c.Enc() != nil {
		EncodeSliceOfDynamicBytesOffset(c.Enc(), *blobs)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfDynamicBytesOffset(c.Dec(), blobs)
		return
	}
	HashSliceOfDynamicBytes(c.Has(), *blobs, maxItems, maxSize)
}

// DefineSliceOfDynamicBytesContent defines the next field as a dynamic slice of dynamic
// binary blobs.
func DefineSliceOfDynamicBytesContent(c CodecI, blobs *[][]byte, maxItems uint64, maxSize uint64) {
	if c.Enc() != nil {
		EncodeSliceOfDynamicBytesContent(c.Enc(), *blobs)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfDynamicBytesContent(c.Dec(), blobs, maxItems, maxSize)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfStaticObjectsOffset defines the next field as a dynamic slice of static
// ssz objects.
func DefineSliceOfStaticObjectsOffset[T newableStaticObject[U], U any](c CodecI, objects *[]T, maxItems uint64) {
	if c.Enc() != nil {
		EncodeSliceOfStaticObjectsOffset(c.Enc(), *objects)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfStaticObjectsOffset(c.Dec(), objects)
		return
	}
	HashSliceOfStaticObjects(c.Has(), *objects, maxItems)
}

// DefineSliceOfStaticObjectsContent defines the next field as a dynamic slice of static
// ssz objects.
func DefineSliceOfStaticObjectsContent[T newableStaticObject[U], U any](c CodecI, objects *[]T, maxItems uint64) {
	if c.Enc() != nil {
		EncodeSliceOfStaticObjectsContent(c.Enc(), *objects)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfStaticObjectsContent(c.Dec(), objects, maxItems)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfDynamicObjectsOffset defines the next field as a dynamic slice of dynamic
// ssz objects.
func DefineSliceOfDynamicObjectsOffset[T newableDynamicObject[U], U any](c CodecI, objects *[]T, maxItems uint64) {
	if c.Enc() != nil {
		EncodeSliceOfDynamicObjectsOffset(c.Enc(), *objects)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfDynamicObjectsOffset(c.Dec(), objects)
		return
	}
	HashSliceOfDynamicObjects(c.Has(), *objects, maxItems)
}

// DefineSliceOfDynamicObjectsContent defines the next field as a dynamic slice of dynamic
// ssz objects.
func DefineSliceOfDynamicObjectsContent[T newableDynamicObject[U], U any](c CodecI, objects *[]T, maxItems uint64) {
	if c.Enc() != nil {
		EncodeSliceOfDynamicObjectsContent(c.Enc(), *objects)
		return
	}
	if c.Dec() != nil {
		DecodeSliceOfDynamicObjectsContent(c.Dec(), objects, maxItems)
		return
	}
	// No hashing, done at the offset position
}
