// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/bits"
	"unsafe"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/gohashtree"
)

// Some helpers to avoid occasional allocations
var (
	hasherBoolFalse = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherBoolTrue  = []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherUint64Pad = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherZeroChunk = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

// Hash is a Merkle hash of an ssz object.
type Hash [32]byte

// Hasher is an SSZ Merkle Hash Root computer.
type Hasher struct {
	scratch []byte // Scratch space for not-yet-hashed writes

	codec *Codec   // Self-referencing to pass DefineSSZ calls through (API trick)
	buf   [32]byte // Integer conversion buffer
}

// HashBool hashes a boolean.
func HashBool[T ~bool](h *Hasher, v T) {
	if !v {
		h.scratch = append(h.scratch, hasherBoolFalse...)
	} else {
		h.scratch = append(h.scratch, hasherBoolTrue...)
	}
}

// HashUint64 hashes a uint64.
func HashUint64[T ~uint64](h *Hasher, n T) {
	binary.LittleEndian.PutUint64(h.buf[:8], (uint64)(n))
	h.scratch = append(h.scratch, h.buf[:8]...)
	h.scratch = append(h.scratch, hasherUint64Pad...)
}

// HashUint256 hashes a uint256.
//
// Note, a nil pointer is hashed as zero.
func HashUint256(h *Hasher, n *uint256.Int) {
	if n != nil {
		n.MarshalSSZInto(h.buf[:32])
		h.scratch = append(h.scratch, h.buf[:32]...)
	} else {
		h.scratch = append(h.scratch, uint256Zero...)
	}
}

// HashStaticBytes hashes a static binary blob.
//
// The blob is passed by pointer to avoid high stack copy costs and a potential
// escape to the heap.
func HashStaticBytes[T commonBytesLengths](h *Hasher, blob *T) {
	// The code below should have used `blob[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	h.hashBytes(unsafe.Slice(&(*blob)[0], len(*blob)))
}

// HashCheckedStaticBytes hashes a static binary blob.
func HashCheckedStaticBytes(h *Hasher, blob []byte) {
	h.hashBytes(blob)
}

// HashDynamicBytes hashes a dynamic binary blob.
func HashDynamicBytes(h *Hasher, blob []byte, maxSize uint64) {
	pos := len(h.scratch)
	h.scratch = append(h.scratch, blob...)
	h.merkleizeWithMixin(pos, uint64(len(blob)), (maxSize+31)/32)
}

// HashStaticObject hashes a static ssz object.
func HashStaticObject(h *Hasher, obj StaticObject) {
	pos := len(h.scratch)
	obj.DefineSSZ(h.codec)
	h.merkleize(pos)
}

// HashDynamicObject hashes a dynamic ssz object.
func HashDynamicObject(h *Hasher, obj DynamicObject) {
	pos := len(h.scratch)
	obj.DefineSSZ(h.codec)
	h.merkleize(pos)
}

// HashArrayOfBits hashes a static array of (packed) bits.
func HashArrayOfBits[T commonBitsLengths](h *Hasher, bits *T) {
	// The code below should have used `*bits[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	h.hashBytes(unsafe.Slice(&(*bits)[0], len(*bits)))
}

// HashSliceOfBits hashes a dynamic slice of (packed) bits.
func HashSliceOfBits(h *Hasher, bits bitfield.Bitlist, maxBits uint64) {
	h.PutBitlist(bits, maxBits)
}

// HashArrayOfUint64s hashes a static array of uint64s.
//
// The reason the ns is passed by pointer and not by value is to prevent it from
// escaping to the heap (and incurring an allocation) when passing it to the
// hasher.
func HashArrayOfUint64s[T commonUint64sLengths](h *Hasher, ns *T) {
	// The code below should have used `*blob[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	nums := unsafe.Slice(&(*ns)[0], len(*ns))

	pos := len(h.scratch)
	for _, n := range nums {
		binary.LittleEndian.PutUint64(h.buf[:8], n)
		h.scratch = append(h.scratch, h.buf[:8]...)
	}
	h.merkleize(pos)
}

// HashSliceOfUint64s hashes a dynamic slice of uint64s.
func HashSliceOfUint64s[T ~uint64](h *Hasher, ns []T, maxItems uint64) {
	pos := len(h.scratch)
	for _, n := range ns {
		binary.LittleEndian.PutUint64(h.buf[:8], (uint64)(n))
		h.scratch = append(h.scratch, h.buf[:8]...)
	}
	h.merkleizeWithMixin(pos, uint64(len(ns)), (maxItems*8+31)/32)
}

// HashArrayOfStaticBytes hashes a static array of static binary blobs.
//
// The reason the blobs is passed by pointer and not by value is to prevent it
// from escaping to the heap (and incurring an allocation) when passing it to
// the output stream.
func HashArrayOfStaticBytes[T commonBytesArrayLengths[U], U commonBytesLengths](h *Hasher, blobs *T) {
	// The code below should have used `(*blobs)[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	HashUnsafeArrayOfStaticBytes(h, unsafe.Slice(&(*blobs)[0], len(*blobs)))
}

// HashUnsafeArrayOfStaticBytes hashes a static array of static binary blobs.
func HashUnsafeArrayOfStaticBytes[T commonBytesLengths](h *Hasher, blobs []T) {
	pos := len(h.scratch)
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.hashBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.merkleize(pos)
}

// HashCheckedArrayOfStaticBytes hashes a static array of static binary blobs.
func HashCheckedArrayOfStaticBytes[T commonBytesLengths](h *Hasher, blobs []T) {
	pos := len(h.scratch)
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.hashBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.merkleize(pos)
}

// HashSliceOfStaticBytes hashes a dynamic slice of static binary blobs.
func HashSliceOfStaticBytes[T commonBytesLengths](h *Hasher, blobs []T, maxItems uint64) {
	pos := len(h.scratch)
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.hashBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.merkleizeWithMixin(pos, uint64(len(blobs)), maxItems)
}

// HashSliceOfDynamicBytes hashes a dynamic slice of dynamic binary blobs.
func HashSliceOfDynamicBytes(h *Hasher, blobs [][]byte, maxItems uint64, maxSize uint64) {
	pos := len(h.scratch)
	for _, blob := range blobs {
		pos := len(h.scratch)
		h.appendBytesChunks(blob)
		h.merkleizeWithMixin(pos, uint64(len(blob)), (maxSize+31)/32)
	}
	h.merkleizeWithMixin(pos, uint64(len(blobs)), maxItems)
}

// HashSliceOfStaticObjects hashes a dynamic slice of static ssz objects.
func HashSliceOfStaticObjects[T StaticObject](h *Hasher, objects []T, maxItems uint64) {
	pos := len(h.scratch)
	for _, obj := range objects {
		pos := len(h.scratch)
		obj.DefineSSZ(h.codec)
		h.merkleize(pos)
	}
	h.merkleizeWithMixin(pos, uint64(len(objects)), maxItems)
}

// HashSliceOfDynamicObjects hashes a dynamic slice of dynamic ssz objects.
func HashSliceOfDynamicObjects[T DynamicObject](h *Hasher, objects []T, maxItems uint64) {
	pos := len(h.scratch)
	for _, obj := range objects {
		pos := len(h.scratch)
		obj.DefineSSZ(h.codec)
		h.merkleize(pos)
	}
	h.merkleizeWithMixin(pos, uint64(len(objects)), maxItems)
}

// hashBytes either appends the blob to the hasher's scratch space if it's small
// enough to fit into a single chunk, or chunks it up and merkleizes it first.
func (h *Hasher) hashBytes(b []byte) {
	if len(b) <= 32 {
		h.appendBytesChunks(b)
		return
	}
	pos := len(h.scratch)
	h.appendBytesChunks(b)
	h.merkleize(pos)
}

// appendBytesChunks appends the blob padded to the 32 byte chunk size.
func (h *Hasher) appendBytesChunks(blob []byte) {
	h.scratch = append(h.scratch, blob...)
	if rest := len(blob) & 0x1f; rest != 0 {
		h.scratch = append(h.scratch, hasherZeroChunk[:32-rest]...)
	}
}

// hash retrieves the computed hash from the hasher.
func (h *Hasher) hash() [32]byte {
	var hash [32]byte
	copy(hash[:], h.scratch)
	return hash
}

var zeroHashes [65][32]byte
var zeroHashLevels map[string]int

func init() {
	zeroHashLevels = make(map[string]int)
	zeroHashLevels[string(make([]byte, 32))] = 0

	tmp := [64]byte{}
	for i := 0; i < 64; i++ {
		copy(tmp[:32], zeroHashes[i][:])
		copy(tmp[32:], zeroHashes[i][:])
		zeroHashes[i+1] = sha256.Sum256(tmp[:])
		zeroHashLevels[string(zeroHashes[i+1][:])] = i + 1
	}
}

// Reset resets the Hasher obj
func (h *Hasher) Reset() {
	h.scratch = h.scratch[:0]
}

func (h *Hasher) FillUpTo32() {
	// pad zero bytes to the left
	if rest := len(h.scratch) % 32; rest != 0 {
		h.scratch = append(h.scratch, hasherZeroChunk[:32-rest]...)
	}
}

func parseBitlist(dst, buf []byte) ([]byte, uint64) {
	msb := uint8(bits.Len8(buf[len(buf)-1])) - 1
	size := uint64(8*(len(buf)-1) + int(msb))

	dst = append(dst, buf...)
	dst[len(dst)-1] &^= uint8(1 << msb)

	newLen := len(dst)
	for i := len(dst) - 1; i >= 0; i-- {
		if dst[i] != 0x00 {
			break
		}
		newLen = i
	}
	res := dst[:newLen]
	return res, size
}

// PutBitlist appends a ssz bitlist
func (h *Hasher) PutBitlist(bb []byte, maxSize uint64) {
	var size uint64
	tmp := make([]byte, 0, len(bb))
	tmp, size = parseBitlist(tmp, bb)

	// merkleize the content with mix in length
	indx := len(h.scratch)
	h.appendBytesChunks(tmp)
	h.merkleizeWithMixin(indx, size, (maxSize+255)/256)
}

// merkleize hashes everything in the scratch space from the starting position.
func (h *Hasher) merkleize(pos int) {
	// merkleizeImpl will expand the `input` by 32 bytes if some hashing depth
	// hits an odd chunk length. But if we're at the end of `h.scratch` already,
	// appending to `input` will allocate a new buffer, *not* expand `h.scratch`,
	// so the next invocation will realloc, over and over and over. We can pre-
	// emptively cater for that by ensuring that an extra 32 bytes is always
	// available.
	if len(h.scratch) == cap(h.scratch) {
		h.scratch = append(h.scratch, hasherZeroChunk...)
		h.scratch = h.scratch[:len(h.scratch)-len(hasherZeroChunk)]
	}
	input := h.scratch[pos:]

	// merkleize the input
	input = h.merkleizeImpl(input[:0], input, 0)
	h.scratch = append(h.scratch[:pos], input...)
}

// merkleizeWithMixin hashes everything in the scratch space from the starting
// position, also mixing in the size of the dynamic slice of data.
func (h *Hasher) merkleizeWithMixin(pos int, num, limit uint64) {
	h.FillUpTo32()
	input := h.scratch[pos:]

	// merkleize the input
	input = h.merkleizeImpl(input[:0], input, limit)

	binary.LittleEndian.PutUint64(h.buf[:8], num)
	input = append(input, h.buf[:8]...)
	input = append(input, hasherUint64Pad...)

	// input is of the form [<input><size>] of 64 bytes
	gohashtree.HashByteSlice(input, input)
	h.scratch = append(h.scratch[:pos], input[:32]...)
}

func nextPowerOfTwo(v uint64) uint {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return uint(v)
}

func getDepth(d uint64) uint8 {
	if d <= 1 {
		return 0
	}
	i := nextPowerOfTwo(d)
	return 64 - uint8(bits.LeadingZeros(i)) - 1
}

func (h *Hasher) merkleizeImpl(dst []byte, input []byte, limit uint64) []byte {
	// count is the number of 32 byte chunks from the input, after right-padding
	// with zeroes to the next multiple of 32 bytes when the input is not aligned
	// to a multiple of 32 bytes.
	count := uint64((len(input) + 31) / 32)
	if limit == 0 {
		limit = count
	} else if count > limit {
		panic(fmt.Sprintf("BUG: count '%d' higher than limit '%d'", count, limit))
	}

	if limit == 0 {
		return append(dst, hasherZeroChunk...)
	}
	if limit == 1 {
		if count == 1 {
			return append(dst, input[:32]...)
		}
		return append(dst, hasherZeroChunk...)
	}

	depth := getDepth(limit)
	if len(input) == 0 {
		return append(dst, zeroHashes[depth][:]...)
	}

	for i := uint8(0); i < depth; i++ {
		layerLen := len(input) / 32
		oddNodeLength := layerLen%2 == 1

		if oddNodeLength {
			// is odd length
			input = append(input, zeroHashes[i][:]...)
			layerLen++
		}

		outputLen := (layerLen / 2) * 32

		gohashtree.HashByteSlice(input, input)
		input = input[:outputLen]
	}

	return append(dst, input...)
}
