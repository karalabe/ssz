// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"math/big"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
)

// Codec is a unified SSZ encoder and decoder that allows simple structs to
// define their schemas once and have that work for both operations at once
// (with the same speed as explicitly typing them out would, of course).
type Codec struct {
	fork Fork // Context for cross-fork monolith types

	enc *Encoder
	dec *Decoder
	has *Hasher
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
func DefineBool[T ~bool](c *Codec, v *T) {
	if c.enc != nil {
		EncodeBool(c.enc, *v)
		return
	}
	if c.dec != nil {
		DecodeBool(c.dec, v)
		return
	}
	HashBool(c.has, *v)
}

// DefineBoolPointerOnFork defines the next field as a 1 byte boolean if present
// in a fork.
func DefineBoolPointerOnFork[T ~bool](c *Codec, v **T, filter ForkFilter) {
	if c.enc != nil {
		EncodeBoolPointerOnFork(c.enc, *v, filter)
		return
	}
	if c.dec != nil {
		DecodeBoolPointerOnFork(c.dec, v, filter)
		return
	}
	HashBoolPointerOnFork(c.has, *v, filter)
}

// DefineUint8 defines the next field as a uint8.
func DefineUint8[T ~uint8](c *Codec, n *T) {
	if c.enc != nil {
		EncodeUint8(c.enc, *n)
		return
	}
	if c.dec != nil {
		DecodeUint8(c.dec, n)
		return
	}
	HashUint8(c.has, *n)
}

// DefineUint8PointerOnFork defines the next field as a uint8 if present in a fork.
func DefineUint8PointerOnFork[T ~uint8](c *Codec, n **T, filter ForkFilter) {
	if c.enc != nil {
		EncodeUint8PointerOnFork(c.enc, *n, filter)
		return
	}
	if c.dec != nil {
		DecodeUint8PointerOnFork(c.dec, n, filter)
		return
	}
	HashUint8PointerOnFork(c.has, *n, filter)
}

// DefineUint16 defines the next field as a uint16.
func DefineUint16[T ~uint16](c *Codec, n *T) {
	if c.enc != nil {
		EncodeUint16(c.enc, *n)
		return
	}
	if c.dec != nil {
		DecodeUint16(c.dec, n)
		return
	}
	HashUint16(c.has, *n)
}

// DefineUint16PointerOnFork defines the next field as a uint16 if present in a fork.
func DefineUint16PointerOnFork[T ~uint16](c *Codec, n **T, filter ForkFilter) {
	if c.enc != nil {
		EncodeUint16PointerOnFork(c.enc, *n, filter)
		return
	}
	if c.dec != nil {
		DecodeUint16PointerOnFork(c.dec, n, filter)
		return
	}
	HashUint16PointerOnFork(c.has, *n, filter)
}

// DefineUint32 defines the next field as a uint32.
func DefineUint32[T ~uint32](c *Codec, n *T) {
	if c.enc != nil {
		EncodeUint32(c.enc, *n)
		return
	}
	if c.dec != nil {
		DecodeUint32(c.dec, n)
		return
	}
	HashUint32(c.has, *n)
}

// DefineUint32PointerOnFork defines the next field as a uint32 if present in a fork.
func DefineUint32PointerOnFork[T ~uint32](c *Codec, n **T, filter ForkFilter) {
	if c.enc != nil {
		EncodeUint32PointerOnFork(c.enc, *n, filter)
		return
	}
	if c.dec != nil {
		DecodeUint32PointerOnFork(c.dec, n, filter)
		return
	}
	HashUint32PointerOnFork(c.has, *n, filter)
}

// DefineUint64 defines the next field as a uint64.
func DefineUint64[T ~uint64](c *Codec, n *T) {
	if c.enc != nil {
		EncodeUint64(c.enc, *n)
		return
	}
	if c.dec != nil {
		DecodeUint64(c.dec, n)
		return
	}
	HashUint64(c.has, *n)
}

// DefineUint64PointerOnFork defines the next field as a uint64 if present in a fork.
func DefineUint64PointerOnFork[T ~uint64](c *Codec, n **T, filter ForkFilter) {
	if c.enc != nil {
		EncodeUint64PointerOnFork(c.enc, *n, filter)
		return
	}
	if c.dec != nil {
		DecodeUint64PointerOnFork(c.dec, n, filter)
		return
	}
	HashUint64PointerOnFork(c.has, *n, filter)
}

// DefineUint256 defines the next field as a uint256.
func DefineUint256(c *Codec, n **uint256.Int) {
	if c.enc != nil {
		EncodeUint256(c.enc, *n)
		return
	}
	if c.dec != nil {
		DecodeUint256(c.dec, n)
		return
	}
	HashUint256(c.has, *n)
}

// DefineUint256OnFork defines the next field as a uint256 if present in a fork.
func DefineUint256OnFork(c *Codec, n **uint256.Int, filter ForkFilter) {
	if c.enc != nil {
		EncodeUint256OnFork(c.enc, *n, filter)
		return
	}
	if c.dec != nil {
		DecodeUint256OnFork(c.dec, n, filter)
		return
	}
	HashUint256OnFork(c.has, *n, filter) // TODO(karalabe): Interesting bug, duplciate, weird place fails, explore
}

// DefineUint256BigInt defines the next field as a uint256.
func DefineUint256BigInt(c *Codec, n **big.Int) {
	if c.enc != nil {
		EncodeUint256BigInt(c.enc, *n)
		return
	}
	if c.dec != nil {
		DecodeUint256BigInt(c.dec, n)
		return
	}
	HashUint256BigInt(c.has, *n)
}

// DefineUint256BigIntOnFork defines the next field as a uint256 if present in a
// fork.
func DefineUint256BigIntOnFork(c *Codec, n **big.Int, filter ForkFilter) {
	if c.enc != nil {
		EncodeUint256BigIntOnFork(c.enc, *n, filter)
		return
	}
	if c.dec != nil {
		DecodeUint256BigIntOnFork(c.dec, n, filter)
		return
	}
	HashUint256BigIntOnFork(c.has, *n, filter)
}

// DefineStaticBytes defines the next field as static binary blob. This method
// can be used for byte arrays.
func DefineStaticBytes[T commonBytesLengths](c *Codec, blob *T) {
	if c.enc != nil {
		EncodeStaticBytes(c.enc, blob)
		return
	}
	if c.dec != nil {
		DecodeStaticBytes(c.dec, blob)
		return
	}
	HashStaticBytes(c.has, blob)
}

// DefineStaticBytesPointerOnFork defines the next field as static binary blob if present
// in a fork. This method can be used for byte arrays.
func DefineStaticBytesPointerOnFork[T commonBytesLengths](c *Codec, blob **T, filter ForkFilter) {
	if c.enc != nil {
		EncodeStaticBytesPointerOnFork(c.enc, *blob, filter)
		return
	}
	if c.dec != nil {
		DecodeStaticBytesPointerOnFork(c.dec, blob, filter)
		return
	}
	HashStaticBytesPointerOnFork(c.has, *blob, filter)
}

// DefineCheckedStaticBytes defines the next field as static binary blob. This
// method can be used for plain byte slices, which is more expensive, since it
// needs runtime size validation.
func DefineCheckedStaticBytes(c *Codec, blob *[]byte, size uint64) {
	if c.enc != nil {
		EncodeCheckedStaticBytes(c.enc, *blob, size)
		return
	}
	if c.dec != nil {
		DecodeCheckedStaticBytes(c.dec, blob, size)
		return
	}
	HashCheckedStaticBytes(c.has, *blob)
}

// DefineDynamicBytesOffset defines the next field as dynamic binary blob.
func DefineDynamicBytesOffset(c *Codec, blob *[]byte, maxSize uint64) {
	if c.enc != nil {
		EncodeDynamicBytesOffset(c.enc, *blob)
		return
	}
	if c.dec != nil {
		DecodeDynamicBytesOffset(c.dec, blob)
		return
	}
	HashDynamicBytes(c.has, *blob, maxSize)
}

// DefineDynamicBytesOffsetOnFork defines the next field as dynamic binary blob
// if present in a fork.
func DefineDynamicBytesOffsetOnFork(c *Codec, blob *[]byte, maxSize uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeDynamicBytesOffsetOnFork(c.enc, *blob, filter)
		return
	}
	if c.dec != nil {
		DecodeDynamicBytesOffsetOnFork(c.dec, blob, filter)
		return
	}
	HashDynamicBytesOnFork(c.has, *blob, maxSize, filter)
}

// DefineDynamicBytesContent defines the next field as dynamic binary blob.
func DefineDynamicBytesContent(c *Codec, blob *[]byte, maxSize uint64) {
	if c.enc != nil {
		EncodeDynamicBytesContent(c.enc, *blob)
		return
	}
	if c.dec != nil {
		DecodeDynamicBytesContent(c.dec, blob, maxSize)
		return
	}
	// No hashing, done at the offset position
}

// DefineDynamicBytesContentOnFork defines the next field as dynamic binary blob
// if present in a fork.
func DefineDynamicBytesContentOnFork(c *Codec, blob *[]byte, maxSize uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeDynamicBytesContentOnFork(c.enc, *blob, filter)
		return
	}
	if c.dec != nil {
		DecodeDynamicBytesContentOnFork(c.dec, blob, maxSize, filter)
		return
	}
	// No hashing, done at the offset position
}

// DefineStaticObject defines the next field as a static ssz object.
func DefineStaticObject[T newableStaticObject[U], U any](c *Codec, obj *T) {
	if c.enc != nil {
		EncodeStaticObject(c.enc, *obj)
		return
	}
	if c.dec != nil {
		DecodeStaticObject(c.dec, obj)
		return
	}
	HashStaticObject(c.has, *obj)
}

// DefineStaticObjectOnFork defines the next field as a static ssz object if
// present in a fork.
func DefineStaticObjectOnFork[T newableStaticObject[U], U any](c *Codec, obj *T, filter ForkFilter) {
	if c.enc != nil {
		EncodeStaticObjectOnFork(c.enc, *obj, filter)
		return
	}
	if c.dec != nil {
		DecodeStaticObjectOnFork(c.dec, obj, filter)
		return
	}
	HashStaticObjectOnFork(c.has, *obj, filter)
}

// DefineDynamicObjectOffset defines the next field as a dynamic ssz object.
func DefineDynamicObjectOffset[T newableDynamicObject[U], U any](c *Codec, obj *T) {
	if c.enc != nil {
		EncodeDynamicObjectOffset(c.enc, *obj)
		return
	}
	if c.dec != nil {
		DecodeDynamicObjectOffset(c.dec, obj)
		return
	}
	HashDynamicObject(c.has, *obj)
}

// DefineDynamicObjectOffsetOnFork defines the next field as a dynamic ssz object
// if present in a fork.
func DefineDynamicObjectOffsetOnFork[T newableDynamicObject[U], U any](c *Codec, obj *T, filter ForkFilter) {
	if c.enc != nil {
		EncodeDynamicObjectOffsetOnFork(c.enc, *obj, filter)
		return
	}
	if c.dec != nil {
		DecodeDynamicObjectOffsetOnFork(c.dec, obj, filter)
		return
	}
	HashDynamicObjectOnFork(c.has, *obj, filter)
}

// DefineDynamicObjectContent defines the next field as a dynamic ssz object.
func DefineDynamicObjectContent[T newableDynamicObject[U], U any](c *Codec, obj *T) {
	if c.enc != nil {
		EncodeDynamicObjectContent(c.enc, *obj)
		return
	}
	if c.dec != nil {
		DecodeDynamicObjectContent(c.dec, obj)
		return
	}
	// No hashing, done at the offset position
}

// DefineDynamicObjectContentOnFork defines the next field as a dynamic ssz object
// if present in a fork.
func DefineDynamicObjectContentOnFork[T newableDynamicObject[U], U any](c *Codec, obj *T, filter ForkFilter) {
	if c.enc != nil {
		EncodeDynamicObjectContentOnFork(c.enc, *obj, filter)
		return
	}
	if c.dec != nil {
		DecodeDynamicObjectContentOnFork(c.dec, obj, filter)
		return
	}
	// No hashing, done at the offset position
}

// DefineArrayOfBits defines the next field as a static array of (packed) bits.
func DefineArrayOfBits[T commonBitsLengths](c *Codec, bits *T, size uint64) {
	if c.enc != nil {
		EncodeArrayOfBits(c.enc, bits)
		return
	}
	if c.dec != nil {
		DecodeArrayOfBits(c.dec, bits, size)
		return
	}
	HashArrayOfBits(c.has, bits)
}

// DefineSliceOfBitsOffset defines the next field as a dynamic slice of (packed)
// bits.
func DefineSliceOfBitsOffset(c *Codec, bits *bitfield.Bitlist, maxBits uint64) {
	if c.enc != nil {
		EncodeSliceOfBitsOffset(c.enc, *bits)
		return
	}
	if c.dec != nil {
		DecodeSliceOfBitsOffset(c.dec, bits)
		return
	}
	HashSliceOfBits(c.has, *bits, maxBits)
}

// DefineSliceOfBitsOffsetOnFork defines the next field as a dynamic slice of
// (packed) bits if present in a fork.
func DefineSliceOfBitsOffsetOnFork(c *Codec, bits *bitfield.Bitlist, maxBits uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfBitsOffsetOnFork(c.enc, *bits, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfBitsOffsetOnFork(c.dec, bits, filter)
		return
	}
	HashSliceOfBitsOnFork(c.has, *bits, maxBits, filter)
}

// DefineSliceOfBitsContent defines the next field as a dynamic slice of (packed)
// bits.
func DefineSliceOfBitsContent(c *Codec, bits *bitfield.Bitlist, maxBits uint64) {
	if c.enc != nil {
		EncodeSliceOfBitsContent(c.enc, *bits)
		return
	}
	if c.dec != nil {
		DecodeSliceOfBitsContent(c.dec, bits, maxBits)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfBitsContentOnFork defines the next field as a dynamic slice of
// (packed) bits if present in a fork.
func DefineSliceOfBitsContentOnFork(c *Codec, bits *bitfield.Bitlist, maxBits uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfBitsContentOnFork(c.enc, *bits, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfBitsContentOnFork(c.dec, bits, maxBits, filter)
		return
	}
	// No hashing, done at the offset position
}

// DefineArrayOfUint64s defines the next field as a static array of uint64s.
func DefineArrayOfUint64s[T commonUint64sLengths](c *Codec, ns *T) {
	if c.enc != nil {
		EncodeArrayOfUint64s(c.enc, ns)
		return
	}
	if c.dec != nil {
		DecodeArrayOfUint64s(c.dec, ns)
		return
	}
	HashArrayOfUint64s(c.has, ns)
}

// DefineSliceOfUint64sOffset defines the next field as a dynamic slice of uint64s.
func DefineSliceOfUint64sOffset[T ~uint64](c *Codec, ns *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfUint64sOffset(c.enc, *ns)
		return
	}
	if c.dec != nil {
		DecodeSliceOfUint64sOffset(c.dec, ns)
		return
	}
	HashSliceOfUint64s(c.has, *ns, maxItems)
}

// DefineSliceOfUint64sOffsetOnFork defines the next field as a dynamic slice of
// uint64s if present in a fork.
func DefineSliceOfUint64sOffsetOnFork[T ~uint64](c *Codec, ns *[]T, maxItems uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfUint64sOffsetOnFork(c.enc, *ns, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfUint64sOffsetOnFork(c.dec, ns, filter)
		return
	}
	HashSliceOfUint64sOnFork(c.has, *ns, maxItems, filter)
}

// DefineSliceOfUint64sContent defines the next field as a dynamic slice of uint64s.
func DefineSliceOfUint64sContent[T ~uint64](c *Codec, ns *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfUint64sContent(c.enc, *ns)
		return
	}
	if c.dec != nil {
		DecodeSliceOfUint64sContent(c.dec, ns, maxItems)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfUint64sContentOnFork defines the next field as a dynamic slice of
// uint64s if present in a fork.
func DefineSliceOfUint64sContentOnFork[T ~uint64](c *Codec, ns *[]T, maxItems uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfUint64sContentOnFork(c.enc, *ns, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfUint64sContentOnFork(c.dec, ns, maxItems, filter)
		return
	}
	// No hashing, done at the offset position
}

// DefineArrayOfStaticBytes defines the next field as a static array of static
// binary blobs.
func DefineArrayOfStaticBytes[T commonBytesArrayLengths[U], U commonBytesLengths](c *Codec, blobs *T) {
	if c.enc != nil {
		EncodeArrayOfStaticBytes[T, U](c.enc, blobs)
		return
	}
	if c.dec != nil {
		DecodeArrayOfStaticBytes[T, U](c.dec, blobs)
		return
	}
	HashArrayOfStaticBytes[T, U](c.has, blobs)
}

// DefineUnsafeArrayOfStaticBytes defines the next field as a static array of
// static binary blobs. This method operates on plain slices of byte arrays and
// will crash if provided a slice of a non-array. Its purpose is to get around
// Go's generics limitations in generated code (use DefineArrayOfStaticBytes).
func DefineUnsafeArrayOfStaticBytes[T commonBytesLengths](c *Codec, blobs []T) {
	if c.enc != nil {
		EncodeUnsafeArrayOfStaticBytes(c.enc, blobs)
		return
	}
	if c.dec != nil {
		DecodeUnsafeArrayOfStaticBytes(c.dec, blobs)
		return
	}
	HashUnsafeArrayOfStaticBytes(c.has, blobs)
}

// DefineCheckedArrayOfStaticBytes defines the next field as a static array of
// static binary blobs. This method can be used for plain slices of byte arrays,
// which is more expensive since it needs runtime size validation.
func DefineCheckedArrayOfStaticBytes[T commonBytesLengths](c *Codec, blobs *[]T, size uint64) {
	if c.enc != nil {
		EncodeCheckedArrayOfStaticBytes(c.enc, *blobs, size)
		return
	}
	if c.dec != nil {
		DecodeCheckedArrayOfStaticBytes(c.dec, blobs, size)
		return
	}
	HashCheckedArrayOfStaticBytes(c.has, *blobs)
}

// DefineSliceOfStaticBytesOffset defines the next field as a dynamic slice of
// static binary blobs.
func DefineSliceOfStaticBytesOffset[T commonBytesLengths](c *Codec, bytes *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfStaticBytesOffset(c.enc, *bytes)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticBytesOffset(c.dec, bytes)
		return
	}
	HashSliceOfStaticBytes(c.has, *bytes, maxItems)
}

// DefineSliceOfStaticBytesOffsetOnFork defines the next field as a dynamic slice
// of static binary blobs if present in a fork.
func DefineSliceOfStaticBytesOffsetOnFork[T commonBytesLengths](c *Codec, bytes *[]T, maxItems uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfStaticBytesOffsetOnFork(c.enc, *bytes, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticBytesOffsetOnFork(c.dec, bytes, filter)
		return
	}
	HashSliceOfStaticBytesOnFork(c.has, *bytes, maxItems, filter)
}

// DefineSliceOfStaticBytesContent defines the next field as a dynamic slice of static
// binary blobs.
func DefineSliceOfStaticBytesContent[T commonBytesLengths](c *Codec, blobs *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfStaticBytesContent(c.enc, *blobs)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticBytesContent(c.dec, blobs, maxItems)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfStaticBytesContentOnFork defines the next field as a dynamic slice
// of static binary blobs if present in a fork.
func DefineSliceOfStaticBytesContentOnFork[T commonBytesLengths](c *Codec, blobs *[]T, maxItems uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfStaticBytesContentOnFork(c.enc, *blobs, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticBytesContentOnFork(c.dec, blobs, maxItems, filter)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfDynamicBytesOffset defines the next field as a dynamic slice of dynamic
// binary blobs.
func DefineSliceOfDynamicBytesOffset(c *Codec, blobs *[][]byte, maxItems uint64, maxSize uint64) {
	if c.enc != nil {
		EncodeSliceOfDynamicBytesOffset(c.enc, *blobs)
		return
	}
	if c.dec != nil {
		DecodeSliceOfDynamicBytesOffset(c.dec, blobs)
		return
	}
	HashSliceOfDynamicBytes(c.has, *blobs, maxItems, maxSize)
}

// DefineSliceOfDynamicBytesContent defines the next field as a dynamic slice of
// dynamic binary blobs.
func DefineSliceOfDynamicBytesContent(c *Codec, blobs *[][]byte, maxItems uint64, maxSize uint64) {
	if c.enc != nil {
		EncodeSliceOfDynamicBytesContent(c.enc, *blobs)
		return
	}
	if c.dec != nil {
		DecodeSliceOfDynamicBytesContent(c.dec, blobs, maxItems, maxSize)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfStaticObjectsOffset defines the next field as a dynamic slice of
// static ssz objects.
func DefineSliceOfStaticObjectsOffset[T newableStaticObject[U], U any](c *Codec, objects *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfStaticObjectsOffset(c.enc, *objects)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticObjectsOffset(c.dec, objects)
		return
	}
	HashSliceOfStaticObjects(c.has, *objects, maxItems)
}

// DefineSliceOfStaticObjectsOffsetOnFork defines the next field as a dynamic
// slice of static ssz objects if present in a fork.
func DefineSliceOfStaticObjectsOffsetOnFork[T newableStaticObject[U], U any](c *Codec, objects *[]T, maxItems uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfStaticObjectsOffsetOnFork(c.enc, *objects, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticObjectsOffsetOnFork(c.dec, objects, filter)
		return
	}
	HashSliceOfStaticObjectsOnFork(c.has, *objects, maxItems, filter)
}

// DefineSliceOfStaticObjectsContent defines the next field as a dynamic slice of static
// ssz objects.
func DefineSliceOfStaticObjectsContent[T newableStaticObject[U], U any](c *Codec, objects *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfStaticObjectsContent(c.enc, *objects)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticObjectsContent(c.dec, objects, maxItems)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfStaticObjectsContentOnFork defines the next field as a dynamic
// slice of static ssz objects if present in a fork.
func DefineSliceOfStaticObjectsContentOnFork[T newableStaticObject[U], U any](c *Codec, objects *[]T, maxItems uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfStaticObjectsContentOnFork(c.enc, *objects, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticObjectsContentOnFork(c.dec, objects, maxItems, filter)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfDynamicObjectsOffset defines the next field as a dynamic slice of
// dynamic ssz objects.
func DefineSliceOfDynamicObjectsOffset[T newableDynamicObject[U], U any](c *Codec, objects *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfDynamicObjectsOffset(c.enc, *objects)
		return
	}
	if c.dec != nil {
		DecodeSliceOfDynamicObjectsOffset(c.dec, objects)
		return
	}
	HashSliceOfDynamicObjects(c.has, *objects, maxItems)
}

// DefineSliceOfDynamicObjectsOffsetOnFork defines the next field as a dynamic
// slice of dynamic ssz objects if present in a fork.
func DefineSliceOfDynamicObjectsOffsetOnFork[T newableDynamicObject[U], U any](c *Codec, objects *[]T, maxItems uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfDynamicObjectsOffsetOnFork(c.enc, *objects, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfDynamicObjectsOffsetOnFork(c.dec, objects, filter)
		return
	}
	HashSliceOfDynamicObjectsOnFork(c.has, *objects, maxItems, filter)
}

// DefineSliceOfDynamicObjectsContent defines the next field as a dynamic slice
// of dynamic ssz objects.
func DefineSliceOfDynamicObjectsContent[T newableDynamicObject[U], U any](c *Codec, objects *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfDynamicObjectsContent(c.enc, *objects)
		return
	}
	if c.dec != nil {
		DecodeSliceOfDynamicObjectsContent(c.dec, objects, maxItems)
		return
	}
	// No hashing, done at the offset position
}

// DefineSliceOfDynamicObjectsContentOnFork defines the next field as a dynamic
// slice of dynamic ssz objects if present in a fork.
func DefineSliceOfDynamicObjectsContentOnFork[T newableDynamicObject[U], U any](c *Codec, objects *[]T, maxItems uint64, filter ForkFilter) {
	if c.enc != nil {
		EncodeSliceOfDynamicObjectsContentOnFork(c.enc, *objects, filter)
		return
	}
	if c.dec != nil {
		DecodeSliceOfDynamicObjectsContentOnFork(c.dec, objects, maxItems, filter)
		return
	}
	// No hashing, done at the offset position
}
