// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"math/bits"
	"unsafe"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
)

// Decoder is a wrapper around an io.Reader or a []byte buffer to implement SSZ
// decoding in a streaming or buffered way. It has the following behaviors:
//
//  1. The decoder does not buffer, simply reads from the wrapped input stream
//     directly. If you need buffering, that is up to you.
//
//  2. The decoder does not return errors that were hit during reading from the
//     underlying input stream from individual encoding methods. Since there
//     is no expectation (in general) for failure, user code can be denser if
//     error checking is done at the end. Internally, of course, an error will
//     halt all future input operations.
//
// Internally there are a few implementation details that maintainers need to be
// aware of when modifying the code:
//
//  1. The decoder supports two modes of operation: streaming and buffered. Any
//     high level Go code would achieve that with two decoder types implementing
//     a common interface. Unfortunately, the DecodeXYZ methods are using Go's
//     generic system, which is not supported on struct/interface *methods*. As
//     such, `Decoder.DecodeUint64s[T ~uint64](ns []T)` style methods cannot be
//     used, only `DecodeUint64s[T ~uint64](end *Decoder, ns []T)`. The latter
//     form then requires each method internally to do some soft of type cast to
//     handle different decoder implementations. To avoid runtime type asserts,
//     we've opted for a combo decoder with 2 possible outputs and switching on
//     which one is set. Elegant? No. Fast? Yes.
//
//  2. A lot of code snippets are repeated (e.g. encoding the offset, which is
//     the exact same for all the different types, yet the code below has them
//     copied verbatim). Unfortunately the Go compiler doesn't inline functions
//     aggressively enough (neither does it allow explicitly directing it to),
//     and in such tight loops, extra calls matter on performance.
type Decoder struct {
	inReader io.Reader // Underlying input stream to read from (streaming mode)
	inRead   uint32    // Bytes already consumed from the reader (streaming mode)
	inReads  []uint32  // Stack of consumed bytes from outer calls (streaming mode)

	inBuffer  []byte    // Underlying input buffer to read from (buffered mode)
	inBufPtr  uintptr   // Starting pointer in the input buffer (buffered mode)
	inBufPtrs []uintptr // Stack of starting pointers from outer calls (buffered mode)
	inBufEnd  uintptr   // Ending pointer in the input buffer (buffered mode)

	err error // Any write error to halt future encoding calls

	codec  *Codec      // Self-referencing to pass DefineSSZ calls through (API trick)
	buf    [32]byte    // Integer conversion buffer
	bufInt uint256.Int // Big.Int conversion buffer (not pointer, alloc free)

	length  uint32   // Message length being decoded
	lengths []uint32 // Stack of lengths from outer calls

	offset  uint32   // Starting offset we expect, or last offset seen after
	offsets []uint32 // Queue of offsets for dynamic size calculations

	sizes  []uint32   // Computed sizes for the dynamic objects
	sizess [][]uint32 // Stack of computed sizes from outer calls
}

// DecodeBool parses a boolean.
func DecodeBool[T ~bool](dec *Decoder, v *T) {
	if dec.err != nil {
		return
	}
	if dec.inReader != nil {
		_, dec.err = io.ReadFull(dec.inReader, dec.buf[:1])
		if dec.err != nil {
			return
		}
		switch dec.buf[0] {
		case 0:
			*v = false
		case 1:
			*v = true
		default:
			dec.err = fmt.Errorf("%w: found %#x", ErrInvalidBoolean, dec.buf[0])
		}
		dec.inRead += 1
	} else {
		if len(dec.inBuffer) < 1 {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		switch dec.inBuffer[0] {
		case 0:
			*v = false
		case 1:
			*v = true
		default:
			dec.err = fmt.Errorf("%w: found %#x", ErrInvalidBoolean, dec.inBuffer[0])
		}
		dec.inBuffer = dec.inBuffer[1:]
	}
}

// DecodeUint64 parses a uint64.
func DecodeUint64[T ~uint64](dec *Decoder, n *T) {
	if dec.err != nil {
		return
	}
	if dec.inReader != nil {
		_, dec.err = io.ReadFull(dec.inReader, dec.buf[:8])
		*n = T(binary.LittleEndian.Uint64(dec.buf[:8]))
		dec.inRead += 8
	} else {
		if len(dec.inBuffer) < 8 {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		*n = T(binary.LittleEndian.Uint64(dec.inBuffer))
		dec.inBuffer = dec.inBuffer[8:]
	}
}

// DecodeUint256 parses a uint256.
func DecodeUint256(dec *Decoder, n **uint256.Int) {
	if dec.err != nil {
		return
	}
	if dec.inReader != nil {
		_, dec.err = io.ReadFull(dec.inReader, dec.buf[:32])
		if dec.err != nil {
			return
		}
		dec.inRead += 32

		if *n == nil {
			*n = new(uint256.Int)
		}
		(*n).UnmarshalSSZ(dec.buf[:32])
	} else {
		if len(dec.inBuffer) < 32 {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		if *n == nil {
			*n = new(uint256.Int)
		}
		(*n).UnmarshalSSZ(dec.inBuffer[:32])
		dec.inBuffer = dec.inBuffer[32:]
	}
}

// DecodeUint256BigInt parses a uint256 into a big.Int.
func DecodeUint256BigInt(dec *Decoder, n **big.Int) {
	if dec.err != nil {
		return
	}
	if dec.inReader != nil {
		_, dec.err = io.ReadFull(dec.inReader, dec.buf[:32])
		if dec.err != nil {
			return
		}
		dec.inRead += 32

		dec.bufInt.UnmarshalSSZ(dec.buf[:32])
		*n = dec.bufInt.ToBig() // TODO(karalabe): make this alloc free (https://github.com/holiman/uint256/pull/177)
	} else {
		if len(dec.inBuffer) < 32 {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		dec.bufInt.UnmarshalSSZ(dec.inBuffer[:32])
		*n = dec.bufInt.ToBig() // TODO(karalabe): make this alloc free (https://github.com/holiman/uint256/pull/177)
		dec.inBuffer = dec.inBuffer[32:]
	}
}

// DecodeStaticBytes parses a static binary blob.
func DecodeStaticBytes[T commonBytesLengths](dec *Decoder, blob *T) {
	if dec.err != nil {
		return
	}
	if dec.inReader != nil {
		// The code below should have used `*blob[:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		_, dec.err = io.ReadFull(dec.inReader, unsafe.Slice(&(*blob)[0], len(*blob)))
		dec.inRead += uint32(len(*blob))
	} else {
		if len(dec.inBuffer) < len(*blob) {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		// The code below should have used `*blob[:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		copy(unsafe.Slice(&(*blob)[0], len(*blob)), dec.inBuffer)
		dec.inBuffer = dec.inBuffer[len(*blob):]
	}
}

// DecodeCheckedStaticBytes parses a static binary blob.
func DecodeCheckedStaticBytes(dec *Decoder, blob *[]byte, size uint64) {
	if dec.err != nil {
		return
	}
	// Expand the byte slice if needed and fill it with the data
	if uint64(cap(*blob)) < size {
		*blob = make([]byte, size)
	} else {
		*blob = (*blob)[:size]
	}
	if dec.inReader != nil {
		_, dec.err = io.ReadFull(dec.inReader, *blob)
		dec.inRead += uint32(size)
	} else {
		if uint64(len(dec.inBuffer)) < size {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		copy(*blob, dec.inBuffer)
		dec.inBuffer = dec.inBuffer[size:]
	}
}

// DecodeDynamicBytesOffset parses a dynamic binary blob.
func DecodeDynamicBytesOffset(dec *Decoder, blob *[]byte) {
	dec.decodeOffset(false)
}

// DecodeDynamicBytesContent is the lazy data reader of DecodeDynamicBytesOffset.
func DecodeDynamicBytesContent(dec *Decoder, blob *[]byte, maxSize uint64) {
	if dec.err != nil {
		return
	}
	// Compute the length of the blob based on the seen offsets
	size := dec.retrieveSize()
	if uint64(size) > maxSize {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxLengthExceeded, size, maxSize)
		return
	}
	// Expand the byte slice if needed and fill it with the data
	if uint32(cap(*blob)) < size {
		*blob = make([]byte, size)
	} else {
		*blob = (*blob)[:size]
	}
	// Inline:
	//
	// DecodeStaticBytes(dec, *(blob))
	if dec.inReader != nil {
		_, dec.err = io.ReadFull(dec.inReader, *blob)
		dec.inRead += size
	} else {
		if uint32(len(dec.inBuffer)) < size {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		copy(*blob, dec.inBuffer)
		dec.inBuffer = dec.inBuffer[size:]
	}
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
	dec.decodeOffset(false)
}

// DecodeDynamicObjectContent is the lazy data reader of DecodeDynamicObjectOffset.
func DecodeDynamicObjectContent[T newableDynamicObject[U], U any](dec *Decoder, obj *T) {
	if dec.err != nil {
		return
	}
	// Compute the length of the object based on the seen offsets
	size := dec.retrieveSize()

	// Descend into a new data slot to track/verify a new sub-length
	dec.descendIntoSlot(size)
	defer dec.ascendFromSlot()

	if *obj == nil {
		*obj = T(new(U))
	}
	dec.startDynamics((*obj).SizeSSZ(true))
	(*obj).DefineSSZ(dec.codec)
	dec.flushDynamics()
}

// DecodeArrayOfBits parses a static array of (packed) bits.
func DecodeArrayOfBits[T commonBitsLengths](dec *Decoder, bits *T, size uint64) {
	if dec.err != nil {
		return
	}
	// The code below should have used `*bits[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	bitvector := unsafe.Slice(&(*bits)[0], len(*bits))

	if dec.inReader != nil {
		_, dec.err = io.ReadFull(dec.inReader, bitvector)
		if dec.err != nil {
			return
		}
		dec.inRead += uint32(len(bitvector))
	} else {
		if len(dec.inBuffer) < len(bitvector) {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		copy(bitvector, dec.inBuffer)
		dec.inBuffer = dec.inBuffer[len(bitvector):]
	}
	// TODO(karalabe): This can probably be done more optimally...
	for i := size; i < uint64(len(bitvector)<<3); i++ {
		if bitvector[i>>3]&(1<<(i&0x7)) > 0 {
			dec.err = fmt.Errorf("%w: bit %d set, size %d bits", ErrJunkInBitvector, i+1, size)
			return
		}
	}
}

// DecodeSliceOfBitsOffset parses a dynamic slice of (packed) bits.
func DecodeSliceOfBitsOffset(dec *Decoder, bitlist *bitfield.Bitlist) {
	dec.decodeOffset(false)
}

// DecodeSliceOfBitsContent is the lazy data reader of DecodeSliceOfBitsOffset.
func DecodeSliceOfBitsContent(dec *Decoder, bitlist *bitfield.Bitlist, maxBits uint64) {
	if dec.err != nil {
		return
	}
	// Compute the length of the encoded bits based on the seen offsets
	size := dec.retrieveSize()
	if size == 0 {
		dec.err = fmt.Errorf("%w: length bit missing", ErrJunkInBitlist)
		return
	}
	// Verify that the byte size is reasonable, bits will need an extra step after decoding
	if maxBytes := maxBits>>3 + 1; maxBytes < uint64(size) {
		dec.err = fmt.Errorf("%w: decoded %d bytes, max %d bytes", ErrMaxItemsExceeded, size, maxBytes)
		return
	}
	// Expand the slice if needed and read the bits
	if uint32(cap(*bitlist)) < size {
		*bitlist = make([]byte, size)
	} else {
		*bitlist = (*bitlist)[:size]
	}
	if dec.inReader != nil {
		_, dec.err = io.ReadFull(dec.inReader, *bitlist)
		if dec.err != nil {
			return
		}
		dec.inRead += uint32(len(*bitlist))
	} else {
		if len(dec.inBuffer) < len(*bitlist) {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		copy(*bitlist, dec.inBuffer)
		dec.inBuffer = dec.inBuffer[len(*bitlist):]
	}
	// Verify that the length bit is at the correct position
	high := (*bitlist)[len(*bitlist)-1]
	if high == 0 {
		dec.err = fmt.Errorf("%w: high byte unset", ErrJunkInBitlist)
		return
	}
	if len := ((len(*bitlist) - 1) >> 3) + bits.Len8(high) - 1; uint64(len) > maxBits {
		dec.err = fmt.Errorf("%w: decoded %d bits, max %d bits", ErrMaxItemsExceeded, len, maxBits)
		return
	}
}

// DecodeArrayOfUint64s parses a static array of uint64s.
func DecodeArrayOfUint64s[T commonUint64sLengths](dec *Decoder, ns *T) {
	if dec.err != nil {
		return
	}
	// The code below should have used `*blob[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	nums := unsafe.Slice(&(*ns)[0], len(*ns))

	if dec.inReader != nil {
		for i := 0; i < len(nums); i++ {
			_, dec.err = io.ReadFull(dec.inReader, dec.buf[:8])
			if dec.err != nil {
				return
			}
			nums[i] = binary.LittleEndian.Uint64(dec.buf[:8])
			dec.inRead += 8
		}
	} else {
		for i := 0; i < len(nums); i++ {
			if len(dec.inBuffer) < 8 {
				dec.err = io.ErrUnexpectedEOF
				return
			}
			nums[i] = binary.LittleEndian.Uint64(dec.inBuffer)
			dec.inBuffer = dec.inBuffer[8:]
		}
	}
}

// DecodeSliceOfUint64sOffset parses a dynamic slice of uint64s.
func DecodeSliceOfUint64sOffset[T ~uint64](dec *Decoder, ns *[]T) {
	dec.decodeOffset(false)
}

// DecodeSliceOfUint64sContent is the lazy data reader of DecodeSliceOfUint64sOffset.
func DecodeSliceOfUint64sContent[T ~uint64](dec *Decoder, ns *[]T, maxItems uint64) {
	if dec.err != nil {
		return
	}
	// Compute the length of the encoded binaries based on the seen offsets
	size := dec.retrieveSize()
	if size == 0 {
		// Empty slice, remove anything extra
		*ns = (*ns)[:0]
		return
	}
	// Compute the number of items based on the item size of the type
	if size&7 != 0 {
		dec.err = fmt.Errorf("%w: length %d, item size %d", ErrDynamicStaticsIndivisible, size, 8)
		return
	}
	itemCount := size >> 3
	if uint64(itemCount) > maxItems {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxItemsExceeded, itemCount, maxItems)
		return
	}
	// Expand the slice if needed and decode the objects
	if uint32(cap(*ns)) < itemCount {
		*ns = make([]T, itemCount)
	} else {
		*ns = (*ns)[:itemCount]
	}
	if dec.inReader != nil {
		for i := uint32(0); i < itemCount; i++ {
			_, dec.err = io.ReadFull(dec.inReader, dec.buf[:8])
			if dec.err != nil {
				return
			}
			(*ns)[i] = T(binary.LittleEndian.Uint64(dec.buf[:8]))
		}
		dec.inRead += 8 * itemCount
	} else {
		for i := uint32(0); i < itemCount; i++ {
			if len(dec.inBuffer) < 8 {
				dec.err = io.ErrUnexpectedEOF
				return
			}
			(*ns)[i] = T(binary.LittleEndian.Uint64(dec.inBuffer))
			dec.inBuffer = dec.inBuffer[8:]
		}
	}
}

// DecodeArrayOfStaticBytes parses a static array of static binary blobs.
func DecodeArrayOfStaticBytes[T commonBytesArrayLengths[U], U commonBytesLengths](dec *Decoder, blobs *T) {
	// The code below should have used `(*blobs)[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	DecodeUnsafeArrayOfStaticBytes(dec, unsafe.Slice(&(*blobs)[0], len(*blobs)))
}

// DecodeUnsafeArrayOfStaticBytes parses a static array of static binary blobs.
func DecodeUnsafeArrayOfStaticBytes[T commonBytesLengths](dec *Decoder, blobs []T) {
	if dec.err != nil {
		return
	}
	if dec.inReader != nil {
		for i := 0; i < len(blobs); i++ {
			// The code below should have used `(*blobs)[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			_, dec.err = io.ReadFull(dec.inReader, unsafe.Slice(&(blobs)[i][0], len((blobs)[i])))
			if dec.err != nil {
				return
			}
			dec.inRead += uint32(len((blobs)[i]))
		}
	} else {
		for i := 0; i < len(blobs); i++ {
			if len(dec.inBuffer) < len((blobs)[i]) {
				dec.err = io.ErrUnexpectedEOF
				return
			}
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			copy(unsafe.Slice(&(blobs)[i][0], len((blobs)[i])), dec.inBuffer)
			dec.inBuffer = dec.inBuffer[len((blobs)[i]):]
		}
	}
}

// DecodeCheckedArrayOfStaticBytes parses a static array of static binary blobs.
func DecodeCheckedArrayOfStaticBytes[T commonBytesLengths](dec *Decoder, blobs *[]T, size uint64) {
	if dec.err != nil {
		return
	}
	// Expand the byte-array slice if needed and fill it with the data
	if uint64(cap(*blobs)) < size {
		*blobs = make([]T, size)
	} else {
		*blobs = (*blobs)[:size]
	}
	if dec.inReader != nil {
		for i := 0; i < len(*blobs); i++ {
			// The code below should have used `(*blobs)[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			_, dec.err = io.ReadFull(dec.inReader, unsafe.Slice(&(*blobs)[i][0], len((*blobs)[i])))
			if dec.err != nil {
				return
			}
			dec.inRead += uint32(len((*blobs)[i]))
		}
	} else {
		for i := 0; i < len(*blobs); i++ {
			if len(dec.inBuffer) < len((*blobs)[i]) {
				dec.err = io.ErrUnexpectedEOF
				return
			}
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			copy(unsafe.Slice(&(*blobs)[i][0], len((*blobs)[i])), dec.inBuffer)
			dec.inBuffer = dec.inBuffer[len((*blobs)[i]):]
		}
	}
}

// DecodeSliceOfStaticBytesOffset parses a dynamic slice of static binary blobs.
func DecodeSliceOfStaticBytesOffset[T commonBytesLengths](dec *Decoder, blobs *[]T) {
	dec.decodeOffset(false)
}

// DecodeSliceOfStaticBytesContent is the lazy data reader of DecodeSliceOfStaticBytesOffset.
func DecodeSliceOfStaticBytesContent[T commonBytesLengths](dec *Decoder, blobs *[]T, maxItems uint64) {
	if dec.err != nil {
		return
	}
	// Compute the length of the encoded binaries based on the seen offsets
	size := dec.retrieveSize()
	if size == 0 {
		// Empty slice, remove anything extra
		*blobs = (*blobs)[:0]
		return
	}
	// Compute the number of items based on the item size of the type
	var sizer T // SizeSSZ is on *U, objects is static, so nil T is fine

	itemSize := uint32(len(sizer))
	if size%itemSize != 0 {
		dec.err = fmt.Errorf("%w: length %d, item size %d", ErrDynamicStaticsIndivisible, size, itemSize)
		return
	}
	itemCount := size / itemSize
	if uint64(itemCount) > maxItems {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxItemsExceeded, itemCount, maxItems)
		return
	}
	// Expand the slice if needed and decode the objects
	if uint32(cap(*blobs)) < itemCount {
		*blobs = make([]T, itemCount)
	} else {
		*blobs = (*blobs)[:itemCount]
	}
	// Descend into a new data slot to track/verify a new sub-length
	dec.descendIntoSlot(size)
	defer dec.ascendFromSlot()

	if dec.inReader != nil {
		for i := uint32(0); i < itemCount; i++ {
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			_, dec.err = io.ReadFull(dec.inReader, unsafe.Slice(&(*blobs)[i][0], len((*blobs)[i])))
			if dec.err != nil {
				return
			}
			dec.inRead += uint32(len((*blobs)[i]))
		}
	} else {
		for i := uint32(0); i < itemCount; i++ {
			if len(dec.inBuffer) < len((*blobs)[i]) {
				dec.err = io.ErrUnexpectedEOF
				return
			}
			// The code below should have used `blobs[i][:]`, alas Go's generics compiler
			// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
			copy(unsafe.Slice(&(*blobs)[i][0], len((*blobs)[i])), dec.inBuffer)
			dec.inBuffer = dec.inBuffer[len((*blobs)[i]):]
		}
	}
}

// DecodeSliceOfDynamicBytesOffset parses a dynamic slice of dynamic binary blobs.
func DecodeSliceOfDynamicBytesOffset(dec *Decoder, blobs *[][]byte) {
	dec.decodeOffset(false)
}

// DecodeSliceOfDynamicBytesContent is the lazy data reader of DecodeSliceOfDynamicBytesOffset.
func DecodeSliceOfDynamicBytesContent(dec *Decoder, blobs *[][]byte, maxItems uint64, maxSize uint64) {
	if dec.err != nil {
		return
	}
	// Compute the length of the blob slice based on the seen offsets and sanity
	// check for empty slice or possibly bad data (too short to encode anything)
	size := dec.retrieveSize()
	if size == 0 {
		// Empty slice, remove anything extra
		*blobs = (*blobs)[:0]
		return
	}
	if size < 4 {
		dec.err = fmt.Errorf("%w: %d bytes available", ErrShortCounterOffset, size)
		return
	}
	// Descend into a new data slot to track/verify a new sub-length
	dec.descendIntoSlot(size)
	defer dec.ascendFromSlot()

	// Since we're decoding a dynamic slice of dynamic objects (blobs here), the
	// first offset will also act as a counter at to how many items there are in
	// the list (x4 bytes for offsets being uint32).
	dec.decodeOffset(true)
	if dec.err != nil {
		return
	}
	if dec.offset == 0 {
		dec.err = ErrZeroCounterOffset
		return
	}
	if dec.offset&3 != 0 {
		dec.err = fmt.Errorf("%w: %d bytes", ErrBadCounterOffset, dec.offsets)
		return
	}
	items := dec.offset >> 2
	if uint64(items) > maxItems {
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
	dec.decodeOffset(false)
}

// DecodeSliceOfStaticObjectsContent is the lazy data reader of DecodeSliceOfStaticObjectsOffset.
func DecodeSliceOfStaticObjectsContent[T newableStaticObject[U], U any](dec *Decoder, objects *[]T, maxItems uint64) {
	if dec.err != nil {
		return
	}
	// Compute the length of the encoded objects based on the seen offsets
	size := dec.retrieveSize()
	if size == 0 {
		// Empty slice, remove anything extra
		*objects = (*objects)[:0]
		return
	}
	// Compute the number of items based on the item size of the type
	var sizer T // SizeSSZ is on *U, objects is static, so nil T is fine

	itemSize := sizer.SizeSSZ()
	if size%itemSize != 0 {
		dec.err = fmt.Errorf("%w: length %d, item size %d", ErrDynamicStaticsIndivisible, size, itemSize)
		return
	}
	itemCount := size / itemSize
	if uint64(itemCount) > maxItems {
		dec.err = fmt.Errorf("%w: decoded %d, max %d", ErrMaxItemsExceeded, itemCount, maxItems)
		return
	}
	// Expand the slice if needed and decode the objects
	if uint32(cap(*objects)) < itemCount {
		*objects = make([]T, itemCount)
	} else {
		*objects = (*objects)[:itemCount]
	}
	// Descend into a new data slot to track/verify a new sub-length
	dec.descendIntoSlot(size)
	defer dec.ascendFromSlot()

	for i := uint32(0); i < itemCount; i++ {
		if (*objects)[i] == nil {
			(*objects)[i] = new(U)
		}
		(*objects)[i].DefineSSZ(dec.codec)
		if dec.err != nil {
			return
		}
	}
}

// DecodeSliceOfDynamicObjectsOffset parses a dynamic slice of dynamic ssz objects.
func DecodeSliceOfDynamicObjectsOffset[T newableDynamicObject[U], U any](dec *Decoder, objects *[]T) {
	dec.decodeOffset(false)
}

// DecodeSliceOfDynamicObjectsContent is the lazy data reader of DecodeSliceOfDynamicObjectsOffset.
func DecodeSliceOfDynamicObjectsContent[T newableDynamicObject[U], U any](dec *Decoder, objects *[]T, maxItems uint64) {
	if dec.err != nil {
		return
	}
	// Compute the length of the blob slice based on the seen offsets and sanity
	// check for empty slice or possibly bad data (too short to encode anything)
	size := dec.retrieveSize()
	if size == 0 {
		// Empty slice, remove anything extra
		*objects = (*objects)[:0]
		return
	}
	if size < 4 {
		dec.err = fmt.Errorf("%w: %d bytes available", ErrShortCounterOffset, size)
		return
	}
	// Descend into a new dynamic list type to track a new sub-length and work
	// with a fresh set of dynamic offsets
	dec.descendIntoSlot(size)
	defer dec.ascendFromSlot()

	// Since we're decoding a dynamic slice of dynamic objects (blobs here), the
	// first offset will also act as a counter at to how many items there are in
	// the list (x4 bytes for offsets being uint32).
	dec.decodeOffset(true)
	if dec.err != nil {
		return
	}
	if dec.offset == 0 {
		dec.err = ErrZeroCounterOffset
		return
	}
	if dec.offset&3 != 0 {
		dec.err = fmt.Errorf("%w: %d bytes", ErrBadCounterOffset, dec.offsets)
		return
	}
	items := dec.offset >> 2
	if uint64(items) > maxItems {
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
func (dec *Decoder) decodeOffset(list bool) {
	if dec.err != nil {
		return
	}
	var offset uint32
	if dec.inReader != nil {
		if _, dec.err = io.ReadFull(dec.inReader, dec.buf[:4]); dec.err != nil {
			return
		}
		offset = binary.LittleEndian.Uint32(dec.buf[:4])
		dec.inRead += 4
	} else {
		if len(dec.inBuffer) < 4 {
			dec.err = io.ErrUnexpectedEOF
			return
		}
		offset = binary.LittleEndian.Uint32(dec.inBuffer)
		dec.inBuffer = dec.inBuffer[4:]
	}
	if offset > dec.length {
		dec.err = fmt.Errorf("%w: decoded %d, message length %d", ErrOffsetBeyondCapacity, offset, dec.length)
		return
	}
	if dec.offsets == nil && !list && dec.offset != offset {
		dec.err = fmt.Errorf("%w: decoded %d, type expects %d", ErrFirstOffsetMismatch, offset, dec.offset)
		return
	}
	if dec.offsets != nil && dec.offset > offset {
		dec.err = fmt.Errorf("%w: decoded %d, previous was %d", ErrBadOffsetProgression, offset, dec.offset)
		return
	}
	dec.offset = offset
	dec.offsets = append(dec.offsets, offset)
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

// descendIntoSlot starts the decoding of a data slot with a new length. For the
// static objects, the length is used to enforce that all data is consumed. For
// the dynamic objects, the length is used to decode the last dynamic item.
func (dec *Decoder) descendIntoSlot(length uint32) {
	dec.lengths = append(dec.lengths, dec.length)
	dec.length = length

	if dec.inReader != nil {
		dec.inReads = append(dec.inReads, dec.inRead)
		dec.inRead = 0
	} else {
		dec.inBufPtrs = append(dec.inBufPtrs, dec.inBufPtr)
		if len(dec.inBuffer) > 0 {
			dec.inBufPtr = uintptr(unsafe.Pointer(&dec.inBuffer[0]))
		} else {
			dec.inBufPtr = dec.inBufEnd // can only happen for bad input
		}
	}
	dec.startDynamics(0) // random offset, will be ignored
}

// ascendFromSlot is the counterpart of descendIntoSlot that enforces the read
// bytes and restores the previously suspended decoding state.
func (dec *Decoder) ascendFromSlot() {
	dec.flushDynamics()

	// For static objects, enforce that the data they read actually corresponds
	// to the data they should have read. Whilst this does not apply to dynamic
	// objects in the current SSZ spec (they will always read all or error with
	// a different issue), there's no reason not to check them for future cases.
	if dec.inReader != nil {
		if dec.inRead != dec.length {
			if dec.err == nil {
				dec.err = fmt.Errorf("%w: data size %d, object consumed %d", ErrObjectSlotSizeMismatch, dec.length, dec.inRead)
			}
		}
		dec.inRead += dec.inReads[len(dec.inReads)-1] // track the sub-reads, don't discard!
		dec.inReads = dec.inReads[:len(dec.inReads)-1]
	} else {
		var read uint32
		if len(dec.inBuffer) > 0 {
			read = uint32(uintptr(unsafe.Pointer(&dec.inBuffer[0])) - dec.inBufPtr)
		} else {
			read = uint32(dec.inBufEnd - dec.inBufPtr)
		}
		if read != dec.length {
			if dec.err == nil {
				dec.err = fmt.Errorf("%w: data size %d, object consumed %d", ErrObjectSlotSizeMismatch, dec.length, read)
			}
		}
		dec.inBufPtr = dec.inBufPtrs[len(dec.inBufPtrs)-1]
		dec.inBufPtrs = dec.inBufPtrs[:len(dec.inBufPtrs)-1]
	}

	dec.length = dec.lengths[len(dec.lengths)-1]
	dec.lengths = dec.lengths[:len(dec.lengths)-1]
}

// startDynamics marks the item being decoded as a dynamic type, setting the starting
// offset for the dynamic fields.
func (dec *Decoder) startDynamics(offset uint32) {
	dec.offset = offset

	// Try to reuse older computed size slices to avoid allocations
	n := len(dec.sizess)

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
}
