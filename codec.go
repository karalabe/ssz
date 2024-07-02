// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import "github.com/holiman/uint256"

// Codec is a unified SSZ encoder, decoder and size that allows simple structs
// to define their schemas once and have that work for all three operations at
// once (with the same speed as explicitly typing them out would, of course)
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

// OffsetDynamics marks the item being encoded as a dynamic type, setting the starting
// offset for the dynamic fields.
func (c *Codec) OffsetDynamics(offset int) func() {
	if c.enc != nil {
		return c.enc.OffsetDynamics(offset)
	}
	return c.dec.OffsetDynamics(offset)
}

// DefineUint64 defines the next field as uint64.
func DefineUint64(c *Codec, n *uint64) {
	if c.enc != nil {
		EncodeUint64(c.enc, *n)
		return
	}
	DecodeUint64(c.dec, n)
}

// DefineUint256 defines the next field as uint256.
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
func DefineSliceOfStaticObjects[T newableObject[U], U any](c *Codec, objects *[]T, maxItems uint32) {
	if c.enc != nil {
		EncodeSliceOfStaticObjects(c, c.enc, *objects)
		return
	}
	DecodeSliceOfStaticObjects(c, c.dec, objects, maxItems)
}
