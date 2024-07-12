// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"crypto/sha256"
	"encoding/binary"
	bitops "math/bits"
	"unsafe"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/gohashtree"
)

// Some helpers to avoid occasional allocations
var (
	hasherBoolFalse = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherBoolTrue  = [32]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherUint64Pad = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	// hasherZeroCache is a pre-computed table of all-zero sub-trie hashing
	hasherZeroCache [65][32]byte
)

func init() {
	var buf [64]byte
	for i := 0; i < len(hasherZeroCache)-1; i++ {
		copy(buf[:32], hasherZeroCache[i][:])
		copy(buf[32:], hasherZeroCache[i][:])

		hasherZeroCache[i+1] = sha256.Sum256(buf[:])
	}
}

// Hasher is an SSZ Merkle Hash Root computer.
type Hasher struct {
	threads bool // Whether threaded hashing is allowed or not

	chunks [][32]byte // Scratch space for in-progress hashing chunks
	depths [][2]int8  // Depth of the individual chunks (layer / chunk)
	layer  int8       // Layer depth being hasher now

	codec  *Codec   // Self-referencing to pass DefineSSZ calls through (API trick)
	buf    [32]byte // Integer conversion buffer
	bitbuf []byte   // Bitlist conversion buffer (
}

// HashBool hashes a boolean.
func HashBool[T ~bool](h *Hasher, v T) {
	if !v {
		h.insertChunk(hasherBoolFalse)
	} else {
		h.insertChunk(hasherBoolTrue)
	}
}

// HashUint64 hashes a uint64.
func HashUint64[T ~uint64](h *Hasher, n T) {
	h.buf = [32]byte{}
	binary.LittleEndian.PutUint64(h.buf[:], (uint64)(n))
	h.insertChunk(h.buf)
}

// HashUint256 hashes a uint256.
//
// Note, a nil pointer is hashed as zero.
func HashUint256(h *Hasher, n *uint256.Int) {
	if n != nil {
		n.MarshalSSZInto(h.buf[:])
		h.insertChunk(h.buf)
	} else {
		h.insertChunk([32]byte{})
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
	h.descendMixinLayer()
	h.insertBlobChunks(blob)
	h.ascendMixinLayer(uint64(len(blob)), (maxSize+31)/32)
}

// HashStaticObject hashes a static ssz object.
func HashStaticObject(h *Hasher, obj StaticObject) {
	h.descendLayer()
	obj.DefineSSZ(h.codec)
	h.ascendLayer(0)
}

// HashDynamicObject hashes a dynamic ssz object.
func HashDynamicObject(h *Hasher, obj DynamicObject) {
	h.descendLayer()
	obj.DefineSSZ(h.codec)
	h.ascendLayer(0)
}

// HashArrayOfBits hashes a static array of (packed) bits.
func HashArrayOfBits[T commonBitsLengths](h *Hasher, bits *T) {
	// The code below should have used `*bits[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	h.hashBytes(unsafe.Slice(&(*bits)[0], len(*bits)))
}

// HashSliceOfBits hashes a dynamic slice of (packed) bits.
func HashSliceOfBits(h *Hasher, bits bitfield.Bitlist, maxBits uint64) {
	// Parse the bit-list into a hashable representation
	var (
		msb  = uint8(bitops.Len8(bits[len(bits)-1])) - 1
		size = uint64((len(bits)-1)<<3 + int(msb))
	)
	h.bitbuf = append(h.bitbuf[:0], bits...)
	h.bitbuf[len(h.bitbuf)-1] &^= uint8(1 << msb)

	newLen := len(h.bitbuf)
	for i := len(h.bitbuf) - 1; i >= 0; i-- {
		if h.bitbuf[i] != 0x00 {
			break
		}
		newLen = i
	}
	h.bitbuf = h.bitbuf[:newLen]

	// Merkleize the content with mixed in length
	h.descendMixinLayer()
	if len(h.bitbuf) == 0 && size > 0 {
		h.insertChunk([32]byte{})
	} else {
		h.insertBlobChunks(h.bitbuf)
	}
	h.ascendMixinLayer(size, (maxBits+255)/256)
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
	h.descendLayer()
	for len(nums) > 4 {
		binary.LittleEndian.PutUint64(h.buf[:], nums[0])
		binary.LittleEndian.PutUint64(h.buf[8:], nums[1])
		binary.LittleEndian.PutUint64(h.buf[16:], nums[2])
		binary.LittleEndian.PutUint64(h.buf[24:], nums[3])

		h.insertChunk(h.buf)
		nums = nums[4:]
	}
	if len(nums) > 0 {
		h.buf = [32]byte{}
		for i := 0; i < len(nums); i++ {
			binary.LittleEndian.PutUint64(h.buf[i<<3:], nums[i])
		}
		h.insertChunk(h.buf)
	}
	h.ascendLayer(0)
}

// HashSliceOfUint64s hashes a dynamic slice of uint64s.
func HashSliceOfUint64s[T ~uint64](h *Hasher, ns []T, maxItems uint64) {
	h.descendMixinLayer()
	nums := ns
	for len(nums) > 4 {
		binary.LittleEndian.PutUint64(h.buf[:], (uint64)(nums[0]))
		binary.LittleEndian.PutUint64(h.buf[8:], (uint64)(nums[1]))
		binary.LittleEndian.PutUint64(h.buf[16:], (uint64)(nums[2]))
		binary.LittleEndian.PutUint64(h.buf[24:], (uint64)(nums[3]))

		h.insertChunk(h.buf)
		nums = nums[4:]
	}
	if len(nums) > 0 {
		h.buf = [32]byte{}
		for i := 0; i < len(nums); i++ {
			binary.LittleEndian.PutUint64(h.buf[i<<3:], (uint64)(nums[i]))
		}
		h.insertChunk(h.buf)
	}
	h.ascendMixinLayer(uint64(len(ns)), (maxItems*8+31)/32)
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
	h.descendLayer()
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.hashBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.ascendLayer(0)
}

// HashCheckedArrayOfStaticBytes hashes a static array of static binary blobs.
func HashCheckedArrayOfStaticBytes[T commonBytesLengths](h *Hasher, blobs []T) {
	h.descendLayer()
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.hashBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.ascendLayer(0)
}

// HashSliceOfStaticBytes hashes a dynamic slice of static binary blobs.
func HashSliceOfStaticBytes[T commonBytesLengths](h *Hasher, blobs []T, maxItems uint64) {
	h.descendMixinLayer()
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.hashBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.ascendMixinLayer(uint64(len(blobs)), maxItems)
}

// HashSliceOfDynamicBytes hashes a dynamic slice of dynamic binary blobs.
func HashSliceOfDynamicBytes(h *Hasher, blobs [][]byte, maxItems uint64, maxSize uint64) {
	h.descendMixinLayer()
	for _, blob := range blobs {
		h.descendMixinLayer()
		h.insertBlobChunks(blob)
		h.ascendMixinLayer(uint64(len(blob)), (maxSize+31)/32)
	}
	h.ascendMixinLayer(uint64(len(blobs)), maxItems)
}

// HashSliceOfStaticObjects hashes a dynamic slice of static ssz objects.
func HashSliceOfStaticObjects[T StaticObject](h *Hasher, objects []T, maxItems uint64) {
	h.descendMixinLayer()
	for _, obj := range objects {
		h.descendLayer()
		obj.DefineSSZ(h.codec)
		h.ascendLayer(0)
	}
	h.ascendMixinLayer(uint64(len(objects)), maxItems)
}

// HashSliceOfDynamicObjects hashes a dynamic slice of dynamic ssz objects.
func HashSliceOfDynamicObjects[T DynamicObject](h *Hasher, objects []T, maxItems uint64) {
	h.descendMixinLayer()
	for _, obj := range objects {
		h.descendLayer()
		obj.DefineSSZ(h.codec)
		h.ascendLayer(0)
	}
	h.ascendMixinLayer(uint64(len(objects)), maxItems)
}

// hashBytes either appends the blob to the hasher's scratch space if it's small
// enough to fit into a single chunk, or chunks it up and merkleizes it first.
func (h *Hasher) hashBytes(blob []byte) {
	// If the blob is small, accumulate as a single chunk
	if len(blob) <= 32 {
		h.buf = [32]byte{}
		copy(h.buf[:], blob)
		h.insertChunk(h.buf)
		return
	}
	// Otherwise hash it as its own tree
	h.descendLayer()
	h.insertBlobChunks(blob)
	h.ascendLayer(0)
}

// insertChunk adds a chunk to the accumulators, collapsing matching pairs.
func (h *Hasher) insertChunk(chunk [32]byte) {
	h.chunks = append(h.chunks, chunk)
	h.depths = append(h.depths, [2]int8{h.layer, 0})
	//fmt.Println("+++", h.depths, "...", h.layer)

	for n := len(h.depths); n > 1 && h.depths[n-1] == h.depths[n-2]; n-- {
		gohashtree.HashChunks(h.chunks[n-2:], h.chunks[n-2:])
		h.depths[n-2][1]++

		h.chunks = h.chunks[:n-1]
		h.depths = h.depths[:n-1]

		//fmt.Println("---", h.depths, "...", h.layer)
	}
}

// insertBlobChunks splits up the blob into 32 byte chunks and adds them to the
// accumulators, collapsing matching pairs.
func (h *Hasher) insertBlobChunks(blob []byte) {
	for len(blob) >= 32 {
		copy(h.buf[:], blob)
		h.insertChunk(h.buf)
		blob = blob[32:]
	}
	if len(blob) > 0 {
		h.buf = [32]byte{}
		copy(h.buf[:], blob)
		h.insertChunk(h.buf)
	}
}

// descendLayer starts a new hashing layer, acting as a barrier to prevent the
// chunks from being collapsed into previous pending ones.
func (h *Hasher) descendLayer() {
	// Descend into the next hashing layer
	h.layer++
	//fmt.Println("^^^", h.depths, "...", h.layer)
}

// ascendLayer terminates a hashing layer, moving the result up one level and
// collapsing anything unblocked. The chunks param controls how many chunks a
// dynamic list is expected to me composed of at maximum (0 == only balance).
func (h *Hasher) ascendLayer(chunks uint64) {
	// If the layer is incomplete, append in zero chunks
	for {
		// If there's only the root chunk left of this layer, check if we've
		// expanded to the required number of chunks and terminate if so.
		n := len(h.depths)
		if (n == 1 || h.depths[n-1][0] != h.depths[n-2][0]) && (1<<h.depths[n-1][1]) >= chunks {
			break
		}
		// Either the layer is not yet balanced, or extensions are needed. Append
		// an empty chunk, collapse it and try again.
		h.chunks = append(h.chunks, hasherZeroCache[h.depths[n-1][1]])
		h.depths = append(h.depths, [2]int8{h.layer, h.depths[n-1][1]})
		//fmt.Println("***", h.depths, "...", h.layer)

		for n := len(h.depths); n > 1 && h.depths[n-1] == h.depths[n-2]; n-- {
			gohashtree.HashChunks(h.chunks[n-2:], h.chunks[n-2:])
			h.depths[n-2][1]++

			h.chunks = h.chunks[:n-1]
			h.depths = h.depths[:n-1]

			//fmt.Println("---", h.depths, "...", h.layer)
		}
	}
	// Ascend from the previous hashing layer
	h.layer--

	n := len(h.depths)
	h.depths[n-1][0]--
	h.depths[n-1][1] = 0
	//fmt.Println("vvv", h.depths, "...", h.layer)

	// Collapse anything that has been unblocks
	for ; n > 1 && h.depths[n-1] == h.depths[n-2]; n-- {
		gohashtree.HashChunks(h.chunks[n-2:], h.chunks[n-2:])
		h.depths[n-2][1]++

		h.chunks = h.chunks[:n-1]
		h.depths = h.depths[:n-1]

		//fmt.Println("---", h.depths)
	}
}

// descendMixinLayer is similar to descendLayer, but actually descends two at the
// same time, using the outer for mixing in a list length during ascent.
func (h *Hasher) descendMixinLayer() {
	h.descendLayer() // length mixin
	h.descendLayer() // data content
}

// ascendMixinLayer is similar to ascendLayer, but actually ascends one for the
// data content, and then mixes in the provided length and ascends once more.
func (h *Hasher) ascendMixinLayer(size uint64, chunks uint64) {
	// If no items have been added, there's nothing to ascend out of. Fix that
	// corner-case here.
	if size == 0 {
		h.insertChunk([32]byte{})
	}
	h.ascendLayer(chunks) // data content

	h.buf = [32]byte{}
	binary.LittleEndian.PutUint64(h.buf[:8], size)
	h.insertChunk(h.buf)

	h.ascendLayer(0) // length mixin
}

// Reset resets the Hasher obj
func (h *Hasher) Reset() {
	h.chunks = h.chunks[:0]
	h.depths = h.depths[:0]
	h.threads = false
}
