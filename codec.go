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

// DefineStaticBytes defines the next field as static binary blob.
func DefineStaticBytes(c *Codec, bytes []byte) {
	if c.enc != nil {
		EncodeStaticBytes(c.enc, bytes)
		return
	}
	DecodeStaticBytes(c.dec, bytes)
}

// DefineDynamicBytes defines the next field as dynamic binary blob.
func DefineDynamicBytes(c *Codec, blob *[]byte, maxSize uint32) {
	if c.enc != nil {
		EncodeDynamicBytes(c.enc, *blob)
		return
	}
	DecodeDynamicBytes(c.dec, blob, maxSize)
}

// DefineStaticObject defines the next field as a static ssz object.
func DefineStaticObject[T newableStaticObject[U], U any](c *Codec, obj *T) {
	if c.enc != nil {
		EncodeStaticObject(c.enc, *obj)
		return
	}
	DecodeStaticObject(c.dec, obj)
}

// DefineDynamicObject defines the next field as a dynamic ssz object.
func DefineDynamicObject[T newableDynamicObject[U], U any](c *Codec, obj *T) {
	if c.enc != nil {
		EncodeDynamicObject(c.enc, *obj)
		return
	}
	DecodeDynamicObject(c.dec, obj)
}

// DefineSliceOfUint64s defines the next field as a dynamic slice of uint64s.
func DefineSliceOfUint64s[T ~uint64](c *Codec, ns *[]T, maxItems uint32) {
	if c.enc != nil {
		EncodeSliceOfUint64s(c.enc, *ns)
		return
	}
	DecodeSliceOfUint64s(c.dec, ns, maxItems)
}

// DefineArrayOfStaticBytes defines the next field as a static array of static
// binary blobs.
func DefineArrayOfStaticBytes[T commonBinaryLengths](c *Codec, bytes []T) {
	if c.enc != nil {
		EncodeArrayOfStaticBytes(c.enc, bytes)
		return
	}
	DecodeArrayOfStaticBytes(c.dec, bytes)
}

// DefineSliceOfStaticBytes defines the next field as a dynamic slice of static
// binary blobs.
func DefineSliceOfStaticBytes[T commonBinaryLengths](c *Codec, bytes *[]T, maxItems uint32) {
	if c.enc != nil {
		EncodeSliceOfStaticBytes(c.enc, *bytes)
		return
	}
	DecodeSliceOfStaticBytes(c.dec, bytes, maxItems)
}

// DefineSliceOfDynamicBytes defines the next field as a dynamic slice of dynamic
// binary blobs.
func DefineSliceOfDynamicBytes(c *Codec, blobs *[][]byte, maxItems uint32, maxSize uint32) {
	if c.enc != nil {
		EncodeSliceOfDynamicBytes(c.enc, *blobs)
		return
	}
	DecodeSliceOfDynamicBytes(c.dec, blobs, maxItems, maxSize)
}

// DefineSliceOfStaticObjects defines the next field as a dynamic slice of static
// ssz objects.
func DefineSliceOfStaticObjects[T newableStaticObject[U], U any](c *Codec, objects *[]T, maxItems uint32) {
	if c.enc != nil {
		EncodeSliceOfStaticObjects(c.enc, *objects)
		return
	}
	DecodeSliceOfStaticObjects(c.dec, objects, maxItems)
}

// DefineSliceOfDynamicObjects defines the next field as a dynamic slice of dynamic
// ssz objects.
func DefineSliceOfDynamicObjects[T newableDynamicObject[U], U any](c *Codec, objects *[]T, maxItems uint32) {
	if c.enc != nil {
		EncodeSliceOfDynamicObjects(c.enc, *objects)
		return
	}
	DecodeSliceOfDynamicObjects(c.dec, objects, maxItems)
}
