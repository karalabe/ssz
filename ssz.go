// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

// Package ssz contains a few coding helpers to implement SSZ codecs.
package ssz

import (
	"io"
)

// Object defines the methods a type needs to implement to be used as an SSZ
// encodable and decodable object.
type Object interface {
	// SizeSSZ returns the total size of an SSZ object.
	SizeSSZ() uint32

	// EncodeSSZ serializes the object though an SSZ encoder.
	EncodeSSZ(enc *Encoder)

	// DecodeSSZ parses the object via an SSZ decoder.
	DecodeSSZ(dec *Decoder)
}

// Encode serializes the provided object into an SSZ stream.
func Encode(w io.Writer, obj Object) error {
	enc := &Encoder{out: w}
	obj.EncodeSSZ(enc)
	return enc.err
}

// Decode parses an object with the given size out of an SSZ stream.
func Decode(r io.Reader, obj Object, size uint32) error {
	dec := &Decoder{in: r, length: size}
	obj.DecodeSSZ(dec)
	return dec.err
}

/*
// decodeSliceLength decodes how many dynamic items are going to occur in the
// stream, capped to the given stream length.
//
// Multiple dynamic items in SSZ are encoded via a list of offsets, followed by
// the list of dynamic items. By looking at the first offset and dividing it by
// 4 bytes (size of an offset), you can derive the number of items in the list.
//
// Note, this method is private as it consumes 4 bytes off the stream and using
// it outside of appropriate measures is non-trivial. Use DecodeSlice instead.
func decodeSliceLength(r io.Reader, limit int) (int, error) {
	// If there are no items at all in the list, return 0
	if limit == 0 {
		return 0, nil
	}
	// Ensure there's at least one offset worth of data
	if limit < 4 {
		return 0, fmt.Errorf("%w: %d bytes available", ErrShortOffset, limit)
	}
	// Decode the offset and convert it into a length
	var offBuf [4]byte
	if _, err := io.ReadFull(r, offBuf[:]); err != nil {
		return 0, err
	}
	length := binary.LittleEndian.Uint32(offBuf[:])
	if length&0xff > 0 {
		return 0, fmt.Errorf("%s: %d bytes", ErrBadCouterOffset, length)
	}
	return int(length >> 2), nil
}
*/
/*
// DecodeSlice decodes a slice of dynamic objects.
//
// It does so by first decoding all the offsets that define the content of the
// slice. The first offset can be used to derive the number of items (content
// starts at first offset, so first offset / 4 bytes == item count). After the
// offsets are decoded, each item is individually decoded based on the offsets.
func DecodeSlice(r io.Reader, kind string, limit int, maxitems int) error {
	// Parse the item count and make sure it's within limits
	items, err := decodeSliceLength(r, limit)
	if err != nil {
		return err
	}
	if items > maxitems {
		return fmt.Errorf("%w: %s has %d items, but onlt %d permitted", ErrMaxItemsExceeded, kind, items, maxitems)
	}
	// Parse all the offsets since we're doing stream processing
	offsets := make([]int, 0, items+1)

}
*/
