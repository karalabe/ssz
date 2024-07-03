// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"

	"github.com/holiman/uint256"
)

// Decoder is a wrapper around an io.Reader to implement dense SSZ decoding. It
// has the following behaviors:
//
//  1. The decoder does not buffer, simply reads from the wrapped input stream
//     directly. If you need buffering, that is up to you.
//
//  2. The decoder does not return errors that were hit during reading from the
//     underlying input stream from individual encoding methods. Since there
//     is no expectation (in general) for failure, user code can be denser if
//     error checking is done at the end. Internally, of course, an error will
//     halt all future input operations.
type Decoder struct {
	in  io.Reader // Underlying output stream to write into
	err error     // Any write error to halt future encoding calls

	codec *Codec   // Self-referencing to pass DefineSSZ calls through (API trick)
	buf   [32]byte // Integer conversion buffer

	length  uint32   // Message length being decoded
	lengths []uint32 // Stack of lengths from outer calls

	offset   uint32     // Starting offset we expect, or last offset seen after
	offsets  []uint32   // Queue of offsets for dynamic size calculations
	offsetss [][]uint32 // Stack of offsets from outer calls

	sizes  []uint32   // Computed sizes for the dynamic objects
	sizess [][]uint32 // Stack of computed sizes from outer calls
}

// DecodeUint64 parses a uint64.
func DecodeUint64[T ~uint64](dec *Decoder, n *T) {
	if dec.err != nil {
		return
	}
	_, dec.err = io.ReadFull(dec.in, dec.buf[:8])
	*n = T(binary.LittleEndian.Uint64(dec.buf[:8]))
}

// DecodeUint256 parses a uint256.
func DecodeUint256(dec *Decoder, n **uint256.Int) {
	if dec.err != nil {
		return
	}
	_, dec.err = io.ReadFull(dec.in, dec.buf[:32])
	if *n == nil {
		*n = new(uint256.Int)
	}
	(*n).UnmarshalSSZ(dec.buf[:32])
}

// DecodeStaticBytes parses a static binary blob.
func DecodeStaticBytes(dec *Decoder, blob []byte) {
	if dec.err != nil {
		return
	}
	_, dec.err = io.ReadFull(dec.in, blob)
}

// DecodeDynamicBytesOffset parses a dynamic binary blob.
func DecodeDynamicBytesOffset(dec *Decoder, blob *[]byte) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
}

// DecodeDynamicBytesContent is the lazy data reader of DecodeDynamicBytesOffset.
func DecodeDynamicBytesContent(dec *Decoder, blob *[]byte, maxSize uint32) {
	if dec.err != nil {
		return
	}
	// Compute the length of the blob based on the seen offsets
	size := dec.retrieveSize()
	if size > maxSize {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxLengthExceeded, size, maxSize)
		return
	}
	// Expand the byte slice if needed and fill it with the data
	if uint32(cap(*blob)) < size {
		*blob = make([]byte, size)
	} else {
		*blob = (*blob)[:size]
	}
	DecodeStaticBytes(dec, *blob)
}

// DecodeStaticObject parses a static ssz object.
func DecodeStaticObject[T newableStaticObject[U], U any](dec *Decoder, obj *T) {
	if dec.err != nil {
		return
	}
	if *obj == nil {
		*obj = T(new(U))
	}
	(*obj).DefineSSZ(dec.codec)
}

// DecodeDynamicObjectOffset parses a dynamic ssz object.
func DecodeDynamicObjectOffset[T newableDynamicObject[U], U any](dec *Decoder, obj *T) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
}

// DecodeDynamicObjectContent is the lazy data reader of DecodeDynamicObjectOffset.
func DecodeDynamicObjectContent[T newableDynamicObject[U], U any](dec *Decoder, obj *T) {
	if dec.err != nil {
		return
	}
	// Compute the length of the object based on the seen offsets
	size := dec.retrieveSize()

	// Descend into a new dynamic list type to track a new sub-length and work
	// with a fresh set of dynamic offsets
	dec.descendIntoDynamic(size)
	defer dec.ascendFromDynamic()

	if *obj == nil {
		*obj = T(new(U))
	}
	dec.startDynamics((*obj).SizeSSZ(true))
	(*obj).DefineSSZ(dec.codec)
	dec.flushDynamics()
}

// DecodeSliceOfUint64sOffset parses a dynamic slice of uint64s.
func DecodeSliceOfUint64sOffset[T ~uint64](dec *Decoder, ns *[]T) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
}

// DecodeSliceOfUint64sContent is the lazy data reader of DecodeSliceOfUint64sOffset.
func DecodeSliceOfUint64sContent[T ~uint64](dec *Decoder, ns *[]T, maxItems uint32) {
	if dec.err != nil {
		return
	}
	// Compute the length of the encoded binaries based on the seen offsets
	size := dec.retrieveSize()
	if size == 0 {
		return // empty slice of objects
	}
	// Compute the number of items based on the item size of the type
	if size%8 != 0 {
		dec.err = fmt.Errorf("%w: length %d, item size %d", ErrDynamicStaticsIndivisible, size, 8)
		return
	}
	itemCount := size / 8
	if itemCount > maxItems {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxItemsExceeded, itemCount, maxItems)
		return
	}
	// Expand the slice if needed and decode the objects
	if uint32(cap(*ns)) < itemCount {
		*ns = make([]T, itemCount)
	} else {
		*ns = (*ns)[:itemCount]
	}
	for i := uint32(0); i < itemCount; i++ {
		DecodeUint64(dec, &(*ns)[i])
	}
}

// DecodeArrayOfStaticBytes parses a static array of static binary blobs.
//
// Note, the input slice is assumed to be pre-allocated.
func DecodeArrayOfStaticBytes[T commonBinaryLengths](dec *Decoder, blobs []T) {
	if dec.err != nil {
		return
	}
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		DecodeStaticBytes(dec, unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
}

// DecodeSliceOfStaticBytesOffset parses a dynamic slice of static binary blobs.
func DecodeSliceOfStaticBytesOffset[T commonBinaryLengths](dec *Decoder, blobs *[]T) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
}

// DecodeSliceOfStaticBytesContent is the lazy data reader of DecodeSliceOfStaticBytesOffset.
func DecodeSliceOfStaticBytesContent[T commonBinaryLengths](dec *Decoder, blobs *[]T, maxItems uint32) {
	if dec.err != nil {
		return
	}
	// Compute the length of the encoded binaries based on the seen offsets
	size := dec.retrieveSize()
	if size == 0 {
		return // empty slice of objects
	}
	// Compute the number of items based on the item size of the type
	var sizer T // SizeSSZ is on *U, objects is static, so nil T is fine

	itemSize := uint32(len(sizer))
	if size%itemSize != 0 {
		dec.err = fmt.Errorf("%w: length %d, item size %d", ErrDynamicStaticsIndivisible, size, itemSize)
		return
	}
	itemCount := size / itemSize
	if itemCount > maxItems {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxItemsExceeded, itemCount, maxItems)
		return
	}
	// Expand the slice if needed and decode the objects
	if uint32(cap(*blobs)) < itemCount {
		*blobs = make([]T, itemCount)
	} else {
		*blobs = (*blobs)[:itemCount]
	}
	for i := uint32(0); i < itemCount; i++ {
		// The code below should have used `blobs[:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		DecodeStaticBytes(dec, unsafe.Slice(&(*blobs)[i][0], len((*blobs)[i])))
	}
}

// DecodeSliceOfDynamicBytesOffset parses a dynamic slice of dynamic binary blobs.
func DecodeSliceOfDynamicBytesOffset(dec *Decoder, blobs *[][]byte) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
}

// DecodeSliceOfDynamicBytesContent is the lazy data reader of DecodeSliceOfDynamicBytesOffset.
func DecodeSliceOfDynamicBytesContent(dec *Decoder, blobs *[][]byte, maxItems uint32, maxSize uint32) {
	if dec.err != nil {
		return
	}
	// Compute the length of the blob slice based on the seen offsets and sanity
	// check for empty slice or possibly bad data (too short to encode anything)
	size := dec.retrieveSize()
	if size == 0 {
		return // empty slice of blobs
	}
	if size < 4 {
		dec.err = fmt.Errorf("%w: %d bytes available", ErrShortCounterOffset, size)
		return
	}
	// Descend into a new dynamic list type to track a new sub-length and work
	// with a fresh set of dynamic offsets
	dec.descendIntoDynamic(size)
	defer dec.ascendFromDynamic()

	// Since we're decoding a dynamic slice of dynamic objects (blobs here), the
	// first offset will also act as a counter at to how many items there are in
	// the list (x4 bytes for offsets being uint32).
	if err := dec.decodeOffset(true); err != nil {
		dec.err = err
		return
	}
	if dec.offset%4 != 0 {
		dec.err = fmt.Errorf("%w: %d bytes", ErrBadCounterOffset, dec.offsets)
		return
	}
	items := dec.offset >> 2
	if items > maxItems {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxItemsExceeded, items, maxItems)
		return
	}
	// Expand the blob slice if needed
	if uint32(cap(*blobs)) < items {
		*blobs = make([][]byte, items)
	} else {
		*blobs = (*blobs)[:items]
	}
	for i := uint32(1); i < items; i++ {
		DecodeDynamicBytesOffset(dec, &(*blobs)[i])
	}
	for i := uint32(0); i < items; i++ {
		DecodeDynamicBytesContent(dec, &(*blobs)[i], maxSize)
	}
}

// DecodeSliceOfStaticObjectsOffset parses a dynamic slice of static ssz objects.
func DecodeSliceOfStaticObjectsOffset[T newableStaticObject[U], U any](dec *Decoder, objects *[]T) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
}

// DecodeSliceOfStaticObjectsContent is the lazy data reader of DecodeSliceOfStaticObjectsOffset.
func DecodeSliceOfStaticObjectsContent[T newableStaticObject[U], U any](dec *Decoder, objects *[]T, maxItems uint32) {
	if dec.err != nil {
		return
	}
	// Compute the length of the encoded objects based on the seen offsets
	size := dec.retrieveSize()
	if size == 0 {
		return // empty slice of objects
	}
	// Compute the number of items based on the item size of the type
	var sizer T // SizeSSZ is on *U, objects is static, so nil T is fine

	itemSize := sizer.SizeSSZ()
	if size%itemSize != 0 {
		dec.err = fmt.Errorf("%w: length %d, item size %d", ErrDynamicStaticsIndivisible, size, itemSize)
		return
	}
	itemCount := size / itemSize
	if itemCount > maxItems {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxItemsExceeded, itemCount, maxItems)
		return
	}
	// Expand the slice if needed and decode the objects
	if uint32(cap(*objects)) < itemCount {
		*objects = make([]T, itemCount)
	} else {
		*objects = (*objects)[:itemCount]
	}
	for i := uint32(0); i < itemCount; i++ {
		if (*objects)[i] == nil {
			(*objects)[i] = new(U)
		}
		(*objects)[i].DefineSSZ(dec.codec)
	}
}

// DecodeSliceOfDynamicObjectsOffset parses a dynamic slice of dynamic ssz objects.
func DecodeSliceOfDynamicObjectsOffset[T newableDynamicObject[U], U any](dec *Decoder, objects *[]T) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
}

// DecodeSliceOfDynamicObjectsContent is the lazy data reader of DecodeSliceOfDynamicObjectsOffset.
func DecodeSliceOfDynamicObjectsContent[T newableDynamicObject[U], U any](dec *Decoder, objects *[]T, maxItems uint32) {
	if dec.err != nil {
		return
	}
	// Compute the length of the blob slice based on the seen offsets and sanity
	// check for empty slice or possibly bad data (too short to encode anything)
	size := dec.retrieveSize()
	if size == 0 {
		return // empty slice of blobs
	}
	if size < 4 {
		dec.err = fmt.Errorf("%w: %d bytes available", ErrShortCounterOffset, size)
		return
	}
	// Descend into a new dynamic list type to track a new sub-length and work
	// with a fresh set of dynamic offsets
	dec.descendIntoDynamic(size)
	defer dec.ascendFromDynamic()

	// Since we're decoding a dynamic slice of dynamic objects (blobs here), the
	// first offset will also act as a counter at to how many items there are in
	// the list (x4 bytes for offsets being uint32).
	if err := dec.decodeOffset(true); err != nil {
		dec.err = err
		return
	}
	if dec.offset%4 != 0 {
		dec.err = fmt.Errorf("%w: %d bytes", ErrBadCounterOffset, dec.offsets)
		return
	}
	items := dec.offset >> 2
	if items > maxItems {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxItemsExceeded, items, maxItems)
		return
	}
	// Expand the blob slice if needed
	if uint32(cap(*objects)) < items {
		*objects = make([]T, items)
	} else {
		*objects = (*objects)[:items]
	}
	for i := uint32(1); i < items; i++ {
		DecodeDynamicObjectOffset(dec, &(*objects)[i])
	}
	for i := uint32(0); i < items; i++ {
		DecodeDynamicObjectContent(dec, &(*objects)[i])
	}
}

// decodeOffset decodes the next uint32 as an offset and validates it.
func (dec *Decoder) decodeOffset(list bool) error {
	if _, err := io.ReadFull(dec.in, dec.buf[:4]); err != nil {
		return err
	}
	offset := binary.LittleEndian.Uint32(dec.buf[:4])
	if offset > dec.length {
		return fmt.Errorf("%w: decoded %d, message length %d", ErrOffsetBeyondCapacity, offset, dec.length)
	}
	if dec.offsets == nil && !list && dec.offset != offset {
		return fmt.Errorf("%w: decoded %d, type expects %d", ErrFirstOffsetMismatch, offset, dec.offset)
	}
	if dec.offsets != nil && dec.offset > offset {
		return fmt.Errorf("%w: decoded %d, previous was %d", ErrBadOffsetProgression, offset, dec.offset)
	}
	dec.offset = offset
	dec.offsets = append(dec.offsets, offset)

	return nil
}

// retrieveSize retrieves the length of the nest dynamic item based on the seen
// and cached offsets.
func (dec *Decoder) retrieveSize() uint32 {
	// If sizes aren't yet available, pre-compute them all. The reason we use a
	// reverse order is to permit popping them off without thrashing the slice.
	if len(dec.sizes) == 0 {
		// Expand the sizes slice to required capacity
		items := len(dec.offsets)
		if cap(dec.sizes) < items {
			dec.sizes = dec.sizes[:cap(dec.sizes)]
			dec.sizes = append(dec.sizes, make([]uint32, items-len(dec.sizes))...)
		} else {
			dec.sizes = dec.sizes[:items]
		}
		// Compute all the sizes we'll need in reverse order (so we can pop them
		// off like a stack without ruining the buffer pointer)
		for i := 0; i < items; i++ {
			if i < items-1 {
				dec.sizes[items-1-i] = dec.offsets[i+1] - dec.offsets[i]
			} else {
				dec.sizes[0] = dec.length - dec.offsets[i]
			}
		}
		// Nuke out the offsets to avoid leaving junk in the state
		dec.offsets = dec.offsets[:0]
	}
	// Retrieve the next item's size and pop it off the size stack
	size := dec.sizes[len(dec.sizes)-1]
	dec.sizes = dec.sizes[:len(dec.sizes)-1]
	return size
}

// descendIntoDynamic is used to trigger the decoding of a new dynamic field with
// a new data length cap.
func (dec *Decoder) descendIntoDynamic(length uint32) {
	dec.lengths = append(dec.lengths, dec.length)
	dec.length = length

	dec.startDynamics(0) // random offset, will be ignored
}

// ascendFromDynamic is the counterpart of descendIntoDynamic that restores the
// previously suspended decoding state.
func (dec *Decoder) ascendFromDynamic() {
	dec.flushDynamics()

	dec.length = dec.lengths[len(dec.lengths)-1]
	dec.lengths = dec.lengths[:len(dec.lengths)-1]
}

// startDynamics marks the item being decoded as a dynamic type, setting the starting
// offset for the dynamic fields.
func (dec *Decoder) startDynamics(offset uint32) {
	// Try to reuse older offset slices to avoid allocations
	n := len(dec.offsetss)

	if cap(dec.offsetss) > n {
		dec.offsetss = dec.offsetss[:n+1]
		dec.offsets, dec.offsetss[n] = dec.offsetss[n], dec.offsets
	} else {
		dec.offsetss = append(dec.offsetss, dec.offsets)
		dec.offsets = nil
	}
	dec.offset = uint32(offset)

	// Try to reuse older computed size slices to avoid allocations
	n = len(dec.sizess)

	if cap(dec.sizess) > n {
		dec.sizess = dec.sizess[:n+1]
		dec.sizes, dec.sizess[n] = dec.sizess[n], dec.sizes
	} else {
		dec.sizess = append(dec.sizess, dec.sizes)
		dec.sizes = nil
	}
}

// flushDynamics marks the end of the dynamic fields, decoding anything queued up and
// restoring any previous states for outer call continuation.
func (dec *Decoder) flushDynamics() {
	// Clear out any leftovers from partial dynamic decodes
	dec.offsets = dec.offsets[:0]
	dec.sizes = dec.sizes[:0]

	// Restore the previous state, but swap in the current slice as a future memcache
	last := len(dec.sizess) - 1

	dec.sizes, dec.sizess[last] = dec.sizess[last], dec.sizes
	dec.sizess = dec.sizess[:last]

	// Restore the previous state, but swap in the current slice as a future memcache
	last = len(dec.offsetss) - 1

	dec.offsets, dec.offsetss[last] = dec.offsetss[last], dec.offsets
	dec.offsetss = dec.offsetss[:last]

	// Note, no need to restore dec.offset. No more new offsets can be found when
	// unrolling the stack and writing out the dynamic data.
}
