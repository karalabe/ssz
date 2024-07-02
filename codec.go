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

// OffsetDynamics marks the item being encoded as a dynamic type, setting the starting
// offset for the dynamic fields.
func (c *Codec) OffsetDynamics(offset int) func() {
	switch {
	case c.enc != nil:
		return c.enc.OffsetDynamics(offset)
	case c.dec != nil:
		return c.dec.OffsetDynamics(offset)
	default:
		panic("not implemented")
	}
}

// DefineUint64 defines the next field as uint64.
func DefineUint64(c *Codec, n *uint64) {
	switch {
	case c.enc != nil:
		EncodeUint64(c.enc, *n)
	case c.dec != nil:
		DecodeUint64(c.dec, n)
	default:
		panic("not implemented")
	}
}

// DefineUint256 defines the next field as uint256.
func DefineUint256(c *Codec, n **uint256.Int) {
	switch {
	case c.enc != nil:
		EncodeUint256(c.enc, *n)
	case c.dec != nil:
		DecodeUint256(c.dec, n)
	default:
		panic("not implemented")
	}
}

// DefineStaticBytes defines the next field as static binary blob.
func DefineStaticBytes(c *Codec, bytes []byte) {
	switch {
	case c.enc != nil:
		EncodeStaticBytes(c.enc, bytes)
	case c.dec != nil:
		DecodeStaticBytes(c.dec, bytes)
	default:
		panic("not implemented")
	}
}

// DefineDynamicBytes defines the next field as dynamic binary blob.
func DefineDynamicBytes(c *Codec, blob *[]byte, maxSize uint32) {
	switch {
	case c.enc != nil:
		EncodeDynamicBytes(c.enc, *blob)
	case c.dec != nil:
		DecodeDynamicBytes(c.dec, blob, maxSize)
	default:
		panic("not implemented")
	}
}

// DefineArrayOfStaticBytes defines the next field as a static array of static
// binary blobs.
func DefineArrayOfStaticBytes[T commonBinaryLengths](c *Codec, bytes []T) {
	switch {
	case c.enc != nil:
		EncodeArrayOfStaticBytes(c.enc, bytes)
	case c.dec != nil:
		DecodeArrayOfStaticBytes(c.dec, bytes)
	default:
		panic("not implemented")
	}
}

// DefineSliceOfStaticBytes defines the next field as a dynamic slice of static
// binary blobs.
func DefineSliceOfStaticBytes[T commonBinaryLengths](c *Codec, bytes *[]T, maxItems uint32) {
	switch {
	case c.enc != nil:
		EncodeSliceOfStaticBytes(c.enc, *bytes)
	case c.dec != nil:
		DecodeSliceOfStaticBytes(c.dec, bytes, maxItems)
	default:
		panic("not implemented")
	}
}

// DefineSliceOfDynamicBytes defines the next field as a dynamic slice of dynamic
// binary blobs.
func DefineSliceOfDynamicBytes(c *Codec, blobs *[][]byte, maxItems uint32, maxSize uint32) {
	switch {
	case c.enc != nil:
		EncodeSliceOfDynamicBytes(c.enc, *blobs)
	case c.dec != nil:
		DecodeSliceOfDynamicBytes(c.dec, blobs, maxItems, maxSize)
	default:
		panic("not implemented")
	}
}

// DefineSliceOfStaticObjects defines the next field as a dynamic slice of static
// ssz objects.
func DefineSliceOfStaticObjects[T newableObject[U], U any](c *Codec, objects *[]T, maxItems uint32) {
	switch {
	case c.enc != nil:
		EncodeSliceOfStaticObjects(c, c.enc, *objects)
	case c.dec != nil:
		DecodeSliceOfStaticObjects(c, c.dec, objects, maxItems)
	default:
		panic("not implemented")
	}
}
