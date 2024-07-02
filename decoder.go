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

	length   uint32     // Message length being decoded
	lengths  []uint32   // Stack of lengths from outer calls
	offset   uint32     // Starting offset we expect, or last offset seen after
	offsets  []uint32   // Queue of offsets for dynamic size calculations
	offsetss [][]uint32 // Stack of offsets from outer calls
	pend     []func()   // Queue of dynamics pending to be decoded
	pends    [][]func() // Stack of dynamics queues from outer calls
}

// OffsetDynamics marks the item being decoded as a dynamic type, setting the starting
// offset for the dynamic fields.
func (dec *Decoder) OffsetDynamics(offset int) {
	dec.offsetss = append(dec.offsetss, dec.offsets)
	dec.offsets = nil
	dec.offset = uint32(offset)
	dec.pends = append(dec.pends, dec.pend)
	dec.pend = nil
}

// FinishDynamics marks the end of the dynamic fields, decoding anything queued up and
// restoring any previous states for outer call continuation.
func (dec *Decoder) FinishDynamics() {
	for _, pend := range dec.pend {
		pend()
	}
	dec.pend = dec.pends[len(dec.pends)-1]
	dec.pends = dec.pends[:len(dec.pends)-1]

	dec.offsets = dec.offsetss[len(dec.offsetss)-1]
	dec.offsetss = dec.offsetss[:len(dec.offsetss)-1]

	// Note, no need to restore dec.offset. No more new offsets can be found when
	// unrolling the stack and writing out the dynamic data.
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

// DecodeDynamicBytes parses a dynamic binary blob.
func DecodeDynamicBytes(dec *Decoder, blob *[]byte, maxSize uint32) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
	dec.pend = append(dec.pend, func() { decodeDynamicBytes(dec, blob, maxSize) })
}

// decodeDynamicBytes is the lazy data reader of DecodeDynamicBytes.
func decodeDynamicBytes(dec *Decoder, blob *[]byte, maxSize uint32) {
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
func DecodeStaticObject[T newableObject[U], U any](dec *Decoder, obj *T) {
	if dec.err != nil {
		return
	}
	if *obj == nil {
		*obj = T(new(U))
	}
	(*obj).DefineSSZ(dec.codec)
}

// DecodeDynamicObject parses a dynamic ssz object.
func DecodeDynamicObject[T newableObject[U], U any](dec *Decoder, obj *T) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
	dec.pend = append(dec.pend, func() { decodeDynamicObject(dec, obj) })
}

// decodeDynamicObject is the lazy data reader of DecodeDynamicObject.
func decodeDynamicObject[T newableObject[U], U any](dec *Decoder, obj *T) {
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
	(*obj).DefineSSZ(dec.codec)
}

// DecodeSliceOfUint64s parses a dynamic slice of uint64s.
func DecodeSliceOfUint64s[T ~uint64](dec *Decoder, ns *[]T, maxItems uint32) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
	dec.pend = append(dec.pend, func() { decodeSliceOfUint64s(dec, ns, maxItems) })
}

// decodeSliceOfUint64s is the lazy data reader of DecodeSliceOfUint64s.
func decodeSliceOfUint64s[T ~uint64](dec *Decoder, ns *[]T, maxItems uint32) {
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

// DecodeSliceOfStaticBytes parses a dynamic slice of static binary blobs.
func DecodeSliceOfStaticBytes[T commonBinaryLengths](dec *Decoder, blobs *[]T, maxItems uint32) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
	dec.pend = append(dec.pend, func() { decodeSliceOfStaticBytes(dec, blobs, maxItems) })
}

// decodeSliceOfStaticBytes is the lazy data reader of DecodeSliceOfStaticBytes.
func decodeSliceOfStaticBytes[T commonBinaryLengths](dec *Decoder, blobs *[]T, maxItems uint32) {
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

// DecodeSliceOfDynamicBytes parses a dynamic slice of dynamic binary blobs.
func DecodeSliceOfDynamicBytes(dec *Decoder, blobs *[][]byte, maxItems uint32, maxSize uint32) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
	dec.pend = append(dec.pend, func() { decodeSliceOfDynamicBytes(dec, blobs, maxItems, maxSize) })
}

// decodeSliceOfDynamicBytes is the lazy data reader of DecodeSliceOfDynamicBytes.
func decodeSliceOfDynamicBytes(dec *Decoder, blobs *[][]byte, maxItems uint32, maxSize uint32) {
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
	// We have consumed the first offset out of bounds, so schedule a dynamic
	// retrieval explicitly for it. For all the rest, consume as individual
	// blobs.
	dec.pend = append(dec.pend, func() { decodeDynamicBytes(dec, &(*blobs)[0], maxSize) })

	for i := uint32(1); i < items; i++ {
		DecodeDynamicBytes(dec, &(*blobs)[i], maxSize)
	}
}

// DecodeSliceOfStaticObjects parses a dynamic slice of static ssz objects.
func DecodeSliceOfStaticObjects[T newableObject[U], U any](dec *Decoder, objects *[]T, maxItems uint32) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
	dec.pend = append(dec.pend, func() { decodeSliceOfStaticObjects(dec, objects, maxItems) })
}

// decodeSliceOfStaticObjects is the lazy data reader of DecodeSliceOfStaticObjects.
func decodeSliceOfStaticObjects[T newableObject[U], U any](dec *Decoder, objects *[]T, maxItems uint32) {
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

// DecodeSliceOfDynamicObjects parses a dynamic slice of dynamic ssz objects.
func DecodeSliceOfDynamicObjects[T newableObject[U], U any](dec *Decoder, objects *[]T, maxItems uint32) {
	if dec.err != nil {
		return
	}
	if dec.err = dec.decodeOffset(false); dec.err != nil {
		return
	}
	dec.pend = append(dec.pend, func() { decodeSliceOfDynamicObjects(dec, objects, maxItems) })
}

// decodeSliceOfDynamicObjects is the lazy data reader of DecodeSliceOfDynamicObjects.
func decodeSliceOfDynamicObjects[T newableObject[U], U any](dec *Decoder, objects *[]T, maxItems uint32) {
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
	// We have consumed the first offset out of bounds, so schedule a dynamic
	// retrieval explicitly for it. For all the rest, consume as individual
	// blobs.
	dec.pend = append(dec.pend, func() { decodeDynamicObject(dec, &(*objects)[0]) })

	for i := uint32(1); i < items; i++ {
		DecodeDynamicObject(dec, &(*objects)[i])
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
	var size uint32

	// If we have many dynamic items, compute the size between them. Otherwise,
	// the last item's size is based on the total message size beinf decoded.
	if len(dec.offsets) > 1 {
		size = dec.offsets[1] - dec.offsets[0]
	} else {
		size = dec.length - dec.offsets[0]
	}
	// Pop off the just-consumed offset and return the size
	dec.offsets = dec.offsets[1:]
	return size
}

// descendIntoDynamic is used to trigger the decoding of a new dynamic field with
// a new data length cap.
func (dec *Decoder) descendIntoDynamic(length uint32) {
	dec.lengths = append(dec.lengths, dec.length)
	dec.length = length

	dec.OffsetDynamics(0) // random offset, will be ignored
}

// ascendFromDynamic is the counterpart of descendIntoDynamic that restores the
// previously suspended decoding state.
func (dec *Decoder) ascendFromDynamic() {
	dec.FinishDynamics()

	dec.length = dec.lengths[len(dec.lengths)-1]
	dec.lengths = dec.lengths[:len(dec.lengths)-1]
}
