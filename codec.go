// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import "github.com/holiman/uint256"

// Codec is a unified SSZ encoder and decoder that allows simple structs to
// define their schemas once and have that work for both operations at once
// (with the same speed as explicitly typing them out would, of course).
type Codec struct {
	enc *Encoder
	dec *Decoder
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

// DefineBool defines the next field as a 1 byte boolean.
func DefineBool[T ~bool](c *Codec, v *T) {
	if c.enc != nil {
		EncodeBool(c.enc, *v)
		return
	}
	DecodeBool(c.dec, v)
}

// DefineUint64 defines the next field as a uint64.
func DefineUint64[T ~uint64](c *Codec, n *T) {
	if c.enc != nil {
		EncodeUint64(c.enc, *n)
		return
	}
	DecodeUint64(c.dec, n)
}

// DefineUint256 defines the next field as a uint256.
func DefineUint256(c *Codec, n **uint256.Int) {
	if c.enc != nil {
		EncodeUint256(c.enc, *n)
		return
	}
	DecodeUint256(c.dec, n)
}

// DefineStaticBytes defines the next field as static binary blob. This method
// can be used for byte arrays.
func DefineStaticBytes(c *Codec, blob []byte) {
	if c.enc != nil {
		EncodeStaticBytes(c.enc, blob)
		return
	}
	DecodeStaticBytes(c.dec, blob)
}

// DefineCheckedStaticBytes defines the next field as static binary blob. This
// method can be used for plain byte slices, which is more expensive , since it
// needs runtime size validation.
func DefineCheckedStaticBytes(c *Codec, blob *[]byte, size uint64) {
	if c.enc != nil {
		EncodeCheckedStaticBytes(c.enc, *blob)
		return
	}
	DecodeCheckedStaticBytes(c.dec, blob, size)
}

// DefineDynamicBytesOffset defines the next field as dynamic binary blob.
func DefineDynamicBytesOffset(c *Codec, blob *[]byte) {
	if c.enc != nil {
		EncodeDynamicBytesOffset(c.enc, *blob)
		return
	}
	DecodeDynamicBytesOffset(c.dec, blob)
}

// DefineDynamicBytesContent defines the next field as dynamic binary blob.
func DefineDynamicBytesContent(c *Codec, blob *[]byte, maxSize uint64) {
	if c.enc != nil {
		EncodeDynamicBytesContent(c.enc, *blob)
		return
	}
	DecodeDynamicBytesContent(c.dec, blob, maxSize)
}

// DefineStaticObject defines the next field as a static ssz object.
func DefineStaticObject[T newableStaticObject[U], U any](c *Codec, obj *T) {
	if c.enc != nil {
		EncodeStaticObject(c.enc, *obj)
		return
	}
	DecodeStaticObject(c.dec, obj)
}

// DefineDynamicObjectOffset defines the next field as a dynamic ssz object.
func DefineDynamicObjectOffset[T newableDynamicObject[U], U any](c *Codec, obj *T) {
	if c.enc != nil {
		EncodeDynamicObjectOffset(c.enc, *obj)
		return
	}
	DecodeDynamicObjectOffset(c.dec, obj)
}

// DefineDynamicObjectContent defines the next field as a dynamic ssz object.
func DefineDynamicObjectContent[T newableDynamicObject[U], U any](c *Codec, obj *T) {
	if c.enc != nil {
		EncodeDynamicObjectContent(c.enc, *obj)
		return
	}
	DecodeDynamicObjectContent(c.dec, obj)
}

// DefineArrayOfUint64s defines the next field as a static array of uint64s.
func DefineArrayOfUint64s[T ~uint64](c *Codec, ns []T) {
	if c.enc != nil {
		EncodeArrayOfUint64s(c.enc, ns)
		return
	}
	DecodeArrayOfUint64s(c.dec, ns)
}

// DefineSliceOfUint64sOffset defines the next field as a dynamic slice of uint64s.
func DefineSliceOfUint64sOffset[T ~uint64](c *Codec, ns *[]T) {
	if c.enc != nil {
		EncodeSliceOfUint64sOffset(c.enc, *ns)
		return
	}
	DecodeSliceOfUint64sOffset(c.dec, ns)
}

// DefineSliceOfUint64sContent defines the next field as a dynamic slice of uint64s.
func DefineSliceOfUint64sContent[T ~uint64](c *Codec, ns *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfUint64sContent(c.enc, *ns)
		return
	}
	DecodeSliceOfUint64sContent(c.dec, ns, maxItems)
}

// DefineArrayOfStaticBytes defines the next field as a static array of static
// binary blobs.
func DefineArrayOfStaticBytes[T commonBytesLengths](c *Codec, blobs []T) {
	if c.enc != nil {
		EncodeArrayOfStaticBytes(c.enc, blobs)
		return
	}
	DecodeArrayOfStaticBytes(c.dec, blobs)
}

// DefineCheckedArrayOfStaticBytes defines the next field as a static array of
// static binary blobs. This method can be used for plain slices of byte arrays,
// which is more expensive  since it needs runtime size validation.
func DefineCheckedArrayOfStaticBytes[T commonBytesLengths](c *Codec, blobs *[]T, size uint64) {
	if c.enc != nil {
		EncodeCheckedArrayOfStaticBytes(c.enc, *blobs)
		return
	}
	DecodeCheckedArrayOfStaticBytes(c.dec, blobs, size)
}

// DefineSliceOfStaticBytesOffset defines the next field as a dynamic slice of static
// binary blobs.
func DefineSliceOfStaticBytesOffset[T commonBytesLengths](c *Codec, bytes *[]T) {
	if c.enc != nil {
		EncodeSliceOfStaticBytesOffset(c.enc, *bytes)
		return
	}
	DecodeSliceOfStaticBytesOffset(c.dec, bytes)
}

// DefineSliceOfStaticBytesContent defines the next field as a dynamic slice of static
// binary blobs.
func DefineSliceOfStaticBytesContent[T commonBytesLengths](c *Codec, blobs *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfStaticBytesContent(c.enc, *blobs)
		return
	}
	DecodeSliceOfStaticBytesContent(c.dec, blobs, maxItems)
}

// DefineSliceOfDynamicBytesOffset defines the next field as a dynamic slice of dynamic
// binary blobs.
func DefineSliceOfDynamicBytesOffset(c *Codec, blobs *[][]byte) {
	if c.enc != nil {
		EncodeSliceOfDynamicBytesOffset(c.enc, *blobs)
		return
	}
	DecodeSliceOfDynamicBytesOffset(c.dec, blobs)
}

// DefineSliceOfDynamicBytesContent defines the next field as a dynamic slice of dynamic
// binary blobs.
func DefineSliceOfDynamicBytesContent(c *Codec, blobs *[][]byte, maxItems uint64, maxSize uint64) {
	if c.enc != nil {
		EncodeSliceOfDynamicBytesContent(c.enc, *blobs)
		return
	}
	DecodeSliceOfDynamicBytesContent(c.dec, blobs, maxItems, maxSize)
}

// DefineSliceOfStaticObjectsOffset defines the next field as a dynamic slice of static
// ssz objects.
func DefineSliceOfStaticObjectsOffset[T newableStaticObject[U], U any](c *Codec, objects *[]T) {
	if c.enc != nil {
		EncodeSliceOfStaticObjectsOffset(c.enc, *objects)
		return
	}
	DecodeSliceOfStaticObjectsOffset(c.dec, objects)
}

// DefineSliceOfStaticObjectsContent defines the next field as a dynamic slice of static
// ssz objects.
func DefineSliceOfStaticObjectsContent[T newableStaticObject[U], U any](c *Codec, objects *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfStaticObjectsContent(c.enc, *objects)
		return
	}
	DecodeSliceOfStaticObjectsContent(c.dec, objects, maxItems)
}

// DefineSliceOfDynamicObjectsOffset defines the next field as a dynamic slice of dynamic
// ssz objects.
func DefineSliceOfDynamicObjectsOffset[T newableDynamicObject[U], U any](c *Codec, objects *[]T) {
	if c.enc != nil {
		EncodeSliceOfDynamicObjectsOffset(c.enc, *objects)
		return
	}
	DecodeSliceOfDynamicObjectsOffset(c.dec, objects)
}

// DefineSliceOfDynamicObjectsContent defines the next field as a dynamic slice of dynamic
// ssz objects.
func DefineSliceOfDynamicObjectsContent[T newableDynamicObject[U], U any](c *Codec, objects *[]T, maxItems uint64) {
	if c.enc != nil {
		EncodeSliceOfDynamicObjectsContent(c.enc, *objects)
		return
	}
	DecodeSliceOfDynamicObjectsContent(c.dec, objects, maxItems)
}
