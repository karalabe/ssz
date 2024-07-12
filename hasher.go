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

// hasherBatch is the number of chunks to batch up before calling the hasher.
const hasherBatch = 8 // *MUST* be power of 2

// Some helpers to avoid occasional allocations
var (
	hasherBoolFalse = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherBoolTrue  = [32]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

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

	chunks [][32]byte   // Scratch space for in-progress hashing chunks
	groups []groupStats // Hashing progress tracking for the chunk groups
	layer  int8         // Layer depth being hasher now

	codec  *Codec // Self-referencing to pass DefineSSZ calls through (API trick)
	bitbuf []byte // Bitlist conversion buffer
}

// groupStats is a metadata structure tracking the stats of a same-level group
// of data chunks waiting to be hashed.
type groupStats struct {
	layer  int8 // Layer this chunk group is from
	depth  int8 // Depth this chunk group is from
	chunks int8 // Number of chunks in this group
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
	var buffer [32]byte
	binary.LittleEndian.PutUint64(buffer[:], uint64(n))
	h.insertChunk(buffer)
}

// HashUint256 hashes a uint256.
//
// Note, a nil pointer is hashed as zero.
func HashUint256(h *Hasher, n *uint256.Int) {
	var buffer [32]byte
	if n != nil {
		n.MarshalSSZInto(buffer[:])
	}
	h.insertChunk(buffer)
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

	var buffer [32]byte
	for len(nums) > 4 {
		binary.LittleEndian.PutUint64(buffer[:], nums[0])
		binary.LittleEndian.PutUint64(buffer[8:], nums[1])
		binary.LittleEndian.PutUint64(buffer[16:], nums[2])
		binary.LittleEndian.PutUint64(buffer[24:], nums[3])

		h.insertChunk(buffer)
		nums = nums[4:]
	}
	if len(nums) > 0 {
		buffer = [32]byte{}
		for i := 0; i < len(nums); i++ {
			binary.LittleEndian.PutUint64(buffer[i<<3:], nums[i])
		}
		h.insertChunk(buffer)
	}
	h.ascendLayer(0)
}

// HashSliceOfUint64s hashes a dynamic slice of uint64s.
func HashSliceOfUint64s[T ~uint64](h *Hasher, ns []T, maxItems uint64) {
	h.descendMixinLayer()
	nums := ns

	var buffer [32]byte
	for len(nums) > 4 {
		binary.LittleEndian.PutUint64(buffer[:], uint64(nums[0]))
		binary.LittleEndian.PutUint64(buffer[8:], uint64(nums[1]))
		binary.LittleEndian.PutUint64(buffer[16:], uint64(nums[2]))
		binary.LittleEndian.PutUint64(buffer[24:], uint64(nums[3]))

		h.insertChunk(buffer)
		nums = nums[4:]
	}
	if len(nums) > 0 {
		buffer = [32]byte{}
		for i := 0; i < len(nums); i++ {
			binary.LittleEndian.PutUint64(buffer[i<<3:], uint64(nums[i]))
		}
		h.insertChunk(buffer)
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
		var buffer [32]byte
		copy(buffer[:], blob)
		h.insertChunk(buffer)
		return
	}
	// Otherwise hash it as its own tree
	h.descendLayer()
	h.insertBlobChunks(blob)
	h.ascendLayer(0)
}

// insertChunk adds a chunk to the accumulators, collapsing matching pairs.
func (h *Hasher) insertChunk(chunk [32]byte) {
	// Insert the chunk into the accumulator
	h.chunks = append(h.chunks, chunk)

	// If the depth tracker is at the leaf level, bump the leaf count
	groups := len(h.groups)
	if groups > 0 && h.groups[groups-1].layer == h.layer && h.groups[groups-1].depth == 0 {
		h.groups[groups-1].chunks++
	} else {
		// New leaf group, create it and early return. Nothing to hash with only
		// one leaf in our chunk list.
		h.groups = append(h.groups, groupStats{
			layer:  h.layer,
			depth:  0,
			chunks: 1,
		})
		return
	}
	// Leaf counter incremented, if not yet enough for a hashing round, return
	group := h.groups[groups-1]
	if group.chunks != hasherBatch {
		return
	}
	for {
		// We've reached **exactly** the batch number of chunks. Note, we're adding
		// them one by one, so can't all of a sudden overshoot. Hash the next batch
		// of chunks and update the trackers.
		chunks := len(h.chunks)
		gohashtree.HashChunks(h.chunks[chunks-hasherBatch:], h.chunks[chunks-hasherBatch:])
		h.chunks = h.chunks[:chunks-hasherBatch/2]

		group.depth++
		group.chunks /= 2

		// The last group tracker we've just hashed needs to be either updated to
		// the new level count, or merged into the previous one if they share all
		// the layer/depth params.
		if groups > 1 {
			prev := h.groups[groups-2]
			if prev.layer == group.layer && prev.depth == group.depth {
				// Two groups can be merged, will trigger a new collapse round
				prev.chunks += group.chunks
				group = prev

				h.groups = h.groups[:groups-1]
				groups--
				continue
			}
		}
		// Either have a single group, or the previous is from a different layer
		// or depth level, update the tail and return
		h.groups[groups-1] = group
		return
	}
}

// insertBlobChunks splits up the blob into 32 byte chunks and adds them to the
// accumulators, collapsing matching pairs.
func (h *Hasher) insertBlobChunks(blob []byte) {
	var buffer [32]byte
	for len(blob) >= 32 {
		copy(buffer[:], blob)
		h.insertChunk(buffer)
		blob = blob[32:]
	}
	if len(blob) > 0 {
		buffer = [32]byte{}
		copy(buffer[:], blob)
		h.insertChunk(buffer)
	}
}

// descendLayer starts a new hashing layer, acting as a barrier to prevent the
// chunks from being collapsed into previous pending ones.
func (h *Hasher) descendLayer() {
	h.layer++
}

// descendMixinLayer is similar to descendLayer, but actually descends two at the
// same time, using the outer for mixing in a list length during ascent.
func (h *Hasher) descendMixinLayer() {
	h.layer += 2
}

// ascendLayer terminates a hashing layer, moving the result up one level and
// collapsing anything unblocked. The capacity param controls how many chunks
// a dynamic list is expected to be composed of at maximum (0 == only balance).
func (h *Hasher) ascendLayer(capacity uint64) {
	// If the layer is incomplete, append in zero chunks. First up, before even
	// caring about maximum length, we must balance the tree (i.e. reduce it to
	// a single root hash).
	for {
		groups := len(h.groups)

		// If the last layer was reduced to one root, we've balanced the tree
		// and can proceed to any expansion if needed
		group := h.groups[groups-1]
		if group.chunks == 1 {
			if groups == 1 || h.groups[groups-2].layer != group.layer {
				break
			}
		}
		// Either group has multiple chunks still, or there are multiple entire
		// groups in this layer. Either way, we need to collapse this group into
		// the previous one and then see.
		if group.chunks&0x1 == 1 {
			// Group unbalanced, expand with a zero sub-trie
			h.chunks = append(h.chunks, hasherZeroCache[group.depth])
			group.chunks++
		}
		chunks := len(h.chunks)
		gohashtree.HashChunks(h.chunks[chunks-int(group.chunks):], h.chunks[chunks-int(group.chunks):])
		h.chunks = h.chunks[:chunks-int(group.chunks)/2]

		group.depth++
		group.chunks /= 2

		// The last group tracker we've just hashed needs to be either updated to
		// the new level count, or merged into the previous one if they share all
		// the layer/depth params.
		if groups > 1 {
			prev := h.groups[groups-2]
			if prev.layer == group.layer && prev.depth == group.depth {
				// Two groups can be merged, may trigger a new collapse round
				h.groups[groups-2].chunks += group.chunks
				h.groups = h.groups[:groups-1]
				continue
			}
		}
		// Either have a single group, or the previous is from a different layer
		// or depth level, update the tail and see if more balancing is needed
		h.groups[groups-1] = group
	}
	// Last group was reduced to a single root hash. If the capacity used during
	// hashing it was less than what the container slot required, keep expanding
	// it with empty sibling tries.
	for {
		groups := len(h.groups)

		// If we've used up the required capacity, stop expanding
		group := h.groups[groups-1]
		if (1 << group.depth) >= capacity {
			break
		}
		// Last group requires expansion, hash in a new empty sibling trie
		h.chunks = append(h.chunks, hasherZeroCache[group.depth])

		chunks := len(h.chunks)
		gohashtree.HashChunks(h.chunks[chunks-2:], h.chunks[chunks-2:])
		h.chunks = h.chunks[:chunks-1]

		h.groups[groups-1].depth++
	}
	// Ascend from the previous hashing layer
	h.layer--

	chunks := len(h.chunks)
	root := h.chunks[chunks-1]
	h.chunks = h.chunks[:chunks-1]

	groups := len(h.groups)
	h.groups = h.groups[:groups-1]

	h.insertChunk(root)
}

// ascendMixinLayer is similar to ascendLayer, but actually ascends one for the
// data content, and then mixes in the provided length and ascends once more.
func (h *Hasher) ascendMixinLayer(size uint64, chunks uint64) {
	// If no items have been added, there's nothing to ascend out of. Fix that
	// corner-case here.
	var buffer [32]byte
	if size == 0 {
		h.insertChunk(buffer)
	}
	h.ascendLayer(chunks) // data content

	binary.LittleEndian.PutUint64(buffer[:8], size)
	h.insertChunk(buffer)

	h.ascendLayer(0) // length mixin
}

// Reset resets the Hasher obj
func (h *Hasher) Reset() {
	h.chunks = h.chunks[:0]
	h.groups = h.groups[:0]
	h.threads = false
}
