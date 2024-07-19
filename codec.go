// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
)

// Codec is a unified SSZ encoder and decoder that allows simple structs to
// define their schemas once and have that work for both operations at once
// (with the same speed as explicitly typing them out would, of course).
type Codec struct {
	enc *Encoder
	dec *Decoder
	has *Hasher
	tre *Treerer
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

// DefineTreerer uses a dedicated treerer in case the types SSZ conversion is for
// some reason asymmetric (e.g. encoding depends on fields, decoding depends on
// outer context).
//
// In reality, it will be the live code run when the object's Merkle tree is being constructed.
func (c *Codec) DefineTreerer(impl func(tre *Treerer)) {
	if c.tre != nil {
		impl(c.tre)
	}
}

// DefineBool defines the next field as a 1 byte boolean.
func DefineBool[T ~bool](c *Codec, v *T) {
	if c.tre != nil {
		TreeifyBool(c.tre, *v)
		return
	}
	if c.enc != nil {
		EncodeBool(c.enc, *v)
		return
	}
	if c.dec != nil {
		DecodeBool(c.dec, v)
		return
	}
	if c.has != nil {
		HashBool(c.has, *v)
	}

}

// DefineUint64 defines the next field as a uint64.
func DefineUint64[T ~uint64](c *Codec, n *T) {
	if c.tre != nil {
		TreeifyUint64(c.tre, *n)
		return
	}
	if c.enc != nil {
		EncodeUint64(c.enc, *n)
		return
	}
	if c.dec != nil {
		DecodeUint64(c.dec, n)
		return
	}
	if c.has != nil {
		HashUint64(c.has, *n)
	}
}

// DefineUint256 defines the next field as a uint256.
func DefineUint256(c *Codec, n **uint256.Int) {
	if c.tre != nil {
		TreeifyUint256(c.tre, *n)
		return
	}
	if c.enc != nil {
		EncodeUint256(c.enc, *n)
		return
	}
	if c.dec != nil {
		DecodeUint256(c.dec, n)
		return
	}
	if c.has != nil {
		HashUint256(c.has, *n)
	}

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
	if c.has != nil {
		HashStaticBytes(c.has, blob)
	}

}

// DefineCheckedStaticBytes defines the next field as static binary blob. This
// method can be used for plain byte slices, which is more expensive, since it
// needs runtime size validation.
func DefineCheckedStaticBytes(c *Codec, blob *[]byte, size uint64) {
	if c.enc != nil {
		EncodeCheckedStaticBytes(c.enc, *blob)
		return
	}
	if c.dec != nil {
		DecodeCheckedStaticBytes(c.dec, blob, size)
		return
	}
	if c.has != nil {
		HashCheckedStaticBytes(c.has, *blob)
	}

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
	if c.has != nil {
		HashDynamicBytes(c.has, *blob, maxSize)
	}
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
	if c.has != nil {
		HashStaticObject(c.has, *obj)
	}
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
	if c.has != nil {
		HashDynamicObject(c.has, *obj)
	}
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
	if c.has != nil {
		HashArrayOfBits(c.has, bits)
	}
}

// DefineSliceOfBitsOffset defines the next field as a dynamic slice of (packed) bits.
func DefineSliceOfBitsOffset(c *Codec, bits *bitfield.Bitlist, maxBits uint64) {
	if c.enc != nil {
		EncodeSliceOfBitsOffset(c.enc, *bits)
		return
	}
	if c.dec != nil {
		DecodeSliceOfBitsOffset(c.dec, bits)
		return
	}
	if c.has != nil {
		HashSliceOfBits(c.has, *bits, maxBits)
	}
}

// DefineSliceOfBitsContent defines the next field as a dynamic slice of (packed) bits.
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
	if c.has != nil {
		HashArrayOfUint64s(c.has, ns)
	}
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
	if c.has != nil {
		HashSliceOfUint64s(c.has, *ns, maxItems)
	}
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
	if c.has != nil {
		HashArrayOfStaticBytes[T, U](c.has, blobs)
	}
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
	if c.has != nil {
		HashUnsafeArrayOfStaticBytes(c.has, blobs)
	}
}

// DefineCheckedArrayOfStaticBytes defines the next field as a static array of
// static binary blobs. This method can be used for plain slices of byte arrays,
// which is more expensive since it needs runtime size validation.
func DefineCheckedArrayOfStaticBytes[T commonBytesLengths](c *Codec, blobs *[]T, size uint64) {
	if c.enc != nil {
		EncodeCheckedArrayOfStaticBytes(c.enc, *blobs)
		return
	}
	if c.dec != nil {
		DecodeCheckedArrayOfStaticBytes(c.dec, blobs, size)
		return
	}
	if c.has != nil {
		HashCheckedArrayOfStaticBytes(c.has, *blobs)
	}
}

// DefineSliceOfStaticBytesOffset defines the next field as a dynamic slice of static
// binary blobs.
func DefineSliceOfStaticBytesOffset[T commonBytesLengths](c *Codec, bytes *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfStaticBytesOffset(c.enc, *bytes)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticBytesOffset(c.dec, bytes)
		return
	}
	if c.has != nil {
		HashSliceOfStaticBytes(c.has, *bytes, maxItems)
	}
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
	if c.has != nil {
		HashSliceOfDynamicBytes(c.has, *blobs, maxItems, maxSize)
	}
}

// DefineSliceOfDynamicBytesContent defines the next field as a dynamic slice of dynamic
// binary blobs.
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

// DefineSliceOfStaticObjectsOffset defines the next field as a dynamic slice of static
// ssz objects.
func DefineSliceOfStaticObjectsOffset[T newableStaticObject[U], U any](c *Codec, objects *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfStaticObjectsOffset(c.enc, *objects)
		return
	}
	if c.dec != nil {
		DecodeSliceOfStaticObjectsOffset(c.dec, objects)
		return
	}
	if c.has != nil {
		HashSliceOfStaticObjects(c.has, *objects, maxItems)
	}
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
	// No hashing, done at the offset posiiton
}

// DefineSliceOfDynamicObjectsOffset defines the next field as a dynamic slice of dynamic
// ssz objects.
func DefineSliceOfDynamicObjectsOffset[T newableDynamicObject[U], U any](c *Codec, objects *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfDynamicObjectsOffset(c.enc, *objects)
		return
	}
	if c.dec != nil {
		DecodeSliceOfDynamicObjectsOffset(c.dec, objects)
		return
	}
	if c.has != nil {
		HashSliceOfDynamicObjects(c.has, *objects, maxItems)
	}
}

// DefineSliceOfDynamicObjectsContent defines the next field as a dynamic slice of dynamic
// ssz objects.
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
