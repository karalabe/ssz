// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	bitops "math/bits"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/gohashtree"
	"golang.org/x/sync/errgroup"
)

// hasherBatch is the number of chunks to batch up before calling the hasher.
const hasherBatch = 8 // *MUST* be power of 2

// concurrencyThreshold is the data size above which a new sub-hasher is spun up
// for each dynamic field instead of hashing sequentially.
const concurrencyThreshold = 65536

// Some helpers to avoid occasional allocations
var (
	hasherZeroChunk = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
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
	layer  int          // Layer depth being hasher now

	codec *Codec // Self-referencing to pass DefineSSZ calls through (API trick)
	sizer *Sizer // Self-referencing to pass SizeSSZ call through (API trick)

	bitbuf []byte // Bitlist conversion buffer
}

// groupStats is a metadata structure tracking the stats of a same-level group
// of data chunks waiting to be hashed.
type groupStats struct {
	layer  int // Layer this chunk group is from
	depth  int // Depth this chunk group is from
	chunks int // Number of chunks in this group
}

// HashBool hashes a boolean.
func HashBool[T ~bool](h *Hasher, v T) {
	if !v {
		h.insertChunk(hasherBoolFalse, 0)
	} else {
		h.insertChunk(hasherBoolTrue, 0)
	}
}

// HashBoolPointerOnFork hashes a boolean if present in a fork.
//
// Note, a nil pointer is hashed as zero.
func HashBoolPointerOnFork[T ~bool](h *Hasher, v *T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	if v == nil {
		HashBool[bool](h, false)
		return
	}
	HashBool(h, *v)
}

// HashUint8 hashes a uint8.
func HashUint8[T ~uint8](h *Hasher, n T) {
	var buffer [32]byte
	buffer[0] = uint8(n)
	h.insertChunk(buffer, 0)
}

// HashUint8PointerOnFork hashes a uint8 if present in a fork.
//
// Note, a nil pointer is hashed as zero.
func HashUint8PointerOnFork[T ~uint8](h *Hasher, n *T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	if n == nil {
		HashUint8[uint8](h, 0)
		return
	}
	HashUint8(h, *n)
}

// HashUint16 hashes a uint16.
func HashUint16[T ~uint16](h *Hasher, n T) {
	var buffer [32]byte
	binary.LittleEndian.PutUint16(buffer[:], uint16(n))
	h.insertChunk(buffer, 0)
}

// HashUint16PointerOnFork hashes a uint16 if present in a fork.
//
// Note, a nil pointer is hashed as zero.
func HashUint16PointerOnFork[T ~uint16](h *Hasher, n *T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	if n == nil {
		HashUint16[uint16](h, 0)
		return
	}
	HashUint16(h, *n)
}

// HashUint32 hashes a uint32.
func HashUint32[T ~uint32](h *Hasher, n T) {
	var buffer [32]byte
	binary.LittleEndian.PutUint32(buffer[:], uint32(n))
	h.insertChunk(buffer, 0)
}

// HashUint32PointerOnFork hashes a uint32 if present in a fork.
//
// Note, a nil pointer is hashed as zero.
func HashUint32PointerOnFork[T ~uint32](h *Hasher, n *T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	if n == nil {
		HashUint32[uint32](h, 0)
		return
	}
	HashUint32(h, *n)
}

// HashUint64 hashes a uint64.
func HashUint64[T ~uint64](h *Hasher, n T) {
	var buffer [32]byte
	binary.LittleEndian.PutUint64(buffer[:], uint64(n))
	h.insertChunk(buffer, 0)
}

// HashUint64PointerOnFork hashes a uint64 if present in a fork.
//
// Note, a nil pointer is hashed as zero.
func HashUint64PointerOnFork[T ~uint64](h *Hasher, n *T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	if n == nil {
		HashUint64[uint64](h, 0)
		return
	}
	HashUint64(h, *n)
}

// HashUint256 hashes a uint256.
//
// Note, a nil pointer is hashed as zero.
func HashUint256(h *Hasher, n *uint256.Int) {
	var buffer [32]byte
	if n != nil {
		n.MarshalSSZInto(buffer[:])
	}
	h.insertChunk(buffer, 0)
}

// HashUint256OnFork hashes a uint256 if present in a fork.
//
// Note, a nil pointer is hashed as zero.
func HashUint256OnFork(h *Hasher, n *uint256.Int, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashUint256(h, n)
}

// HashUint256BigInt hashes a big.Int as uint256.
//
// Note, a nil pointer is hashed as zero.
// Note, an overflow will be silently dropped.
func HashUint256BigInt(h *Hasher, n *big.Int) {
	var buffer [32]byte
	if n != nil {
		var bufint uint256.Int // No pointer, alloc free
		bufint.SetFromBig(n)
		bufint.MarshalSSZInto(buffer[:])
	}
	h.insertChunk(buffer, 0)
}

// HashUint256BigIntOnFork hashes a big.Int as uint256 if present in a fork.
//
// Note, a nil pointer is hashed as zero.
// Note, an overflow will be silently dropped.
func HashUint256BigIntOnFork(h *Hasher, n *big.Int, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashUint256BigInt(h, n)
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

// HashStaticBytesPointerOnFork hashes a static binary blob if present in a fork.
//
// Note, a nil pointer is hashed as an empty binary blob.
func HashStaticBytesPointerOnFork[T commonBytesLengths](h *Hasher, blob *T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	if blob == nil {
		// Go generics cannot do len(T{}), so we either allocate and bear the GC
		// costs, or we use reflect. Both is kind of crappy.
		//
		// https://github.com/golang/go/issues/69100
		h.hashBytesEmpty(reflect.TypeFor[T]().Len())
		return
	}
	HashStaticBytes(h, blob)
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

// HashDynamicBytesOnFork hashes a dynamic binary blob if present in a fork.
func HashDynamicBytesOnFork(h *Hasher, blob []byte, maxSize uint64, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashDynamicBytes(h, blob, maxSize)
}

// HashStaticObject hashes a static ssz object.
func HashStaticObject[T newableStaticObject[U], U any](h *Hasher, obj T) {
	h.descendLayer()
	if obj == nil {
		// If the object is nil, pull up it's zero value. This will be very slow,
		// but it should not happen in production, only during tests mostly.
		obj = zeroValueStatic[T, U]()
	}
	obj.DefineSSZ(h.codec)
	h.ascendLayer(0)
}

// HashStaticObjectOnFork hashes a static ssz object if present in a fork.
func HashStaticObjectOnFork[T newableStaticObject[U], U any](h *Hasher, obj T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashStaticObject(h, obj)
}

// HashDynamicObject hashes a dynamic ssz object.
func HashDynamicObject[T newableDynamicObject[U], U any](h *Hasher, obj T) {
	h.descendLayer()
	if obj == nil {
		// If the object is nil, pull up it's zero value. This will be very slow,
		// but it should not happen in production, only during tests mostly.
		obj = zeroValueDynamic[T, U]()
	}
	obj.DefineSSZ(h.codec)
	h.ascendLayer(0)
}

// HashDynamicObjectOnFork hashes a dynamic ssz object if present in a fork.
func HashDynamicObjectOnFork[T newableDynamicObject[U], U any](h *Hasher, obj T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashDynamicObject(h, obj)
}

// HashArrayOfBits hashes a static array of (packed) bits.
func HashArrayOfBits[T commonBitsLengths](h *Hasher, bits *T) {
	// The code below should have used `*bits[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	h.hashBytes(unsafe.Slice(&(*bits)[0], len(*bits)))
}

// HashArrayOfBitsPointerOnFork hashes a static array of (packed) bits if present
// in a fork.
func HashArrayOfBitsPointerOnFork[T commonBitsLengths](h *Hasher, bits *T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	if bits == nil {
		// Go generics cannot do len(T{}), so we either allocate and bear the GC
		// costs, or we use reflect. Both is kind of crappy.
		//
		// https://github.com/golang/go/issues/69100
		h.hashBytesEmpty(reflect.TypeFor[T]().Len())
		return
	}
	HashArrayOfBits(h, bits)
}

// HashSliceOfBits hashes a dynamic slice of (packed) bits.
//
// Note, a nil slice of bits is serialized as an empty bit list.
func HashSliceOfBits(h *Hasher, bits bitfield.Bitlist, maxBits uint64) {
	// If the slice of bits is nil (i.e. uninitialized), hash it as empty
	if bits == nil {
		HashSliceOfBits(h, bitlistZero, maxBits)
		return
	}
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
		h.insertChunk([32]byte{}, 0)
	} else {
		h.insertBlobChunks(h.bitbuf)
	}
	h.ascendMixinLayer(size, (maxBits+255)/256)
}

// HashSliceOfBitsOnFork hashes a dynamic slice of (packed) bits if present in a
// fork.
//
// Note, a nil slice of bits is serialized as an empty bit list.
func HashSliceOfBitsOnFork(h *Hasher, bits bitfield.Bitlist, maxBits uint64, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashSliceOfBits(h, bits, maxBits)
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

		h.insertChunk(buffer, 0)
		nums = nums[4:]
	}
	if len(nums) > 0 {
		buffer = [32]byte{}
		for i := 0; i < len(nums); i++ {
			binary.LittleEndian.PutUint64(buffer[i<<3:], nums[i])
		}
		h.insertChunk(buffer, 0)
	}
	h.ascendLayer(0)
}

// HashArrayOfUint64sPointerOnFork hashes a static array of uint64s if present
// in a fork.
func HashArrayOfUint64sPointerOnFork[T commonUint64sLengths](h *Hasher, ns *T, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	if ns == nil {
		h.descendLayer()
		h.insertBlobChunksEmpty(reflect.TypeFor[T]().Len() * 8)
		h.ascendLayer(0)
		return
	}
	HashArrayOfUint64s(h, ns)
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

		h.insertChunk(buffer, 0)
		nums = nums[4:]
	}
	if len(nums) > 0 {
		buffer = [32]byte{}
		for i := 0; i < len(nums); i++ {
			binary.LittleEndian.PutUint64(buffer[i<<3:], uint64(nums[i]))
		}
		h.insertChunk(buffer, 0)
	}
	h.ascendMixinLayer(uint64(len(ns)), (maxItems*8+31)/32)
}

// HashSliceOfUint64sOnFork hashes a dynamic slice of uint64s if present in a fork.
func HashSliceOfUint64sOnFork[T ~uint64](h *Hasher, ns []T, maxItems uint64, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashSliceOfUint64s(h, ns, maxItems)
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

// HashSliceOfStaticBytesOnFork hashes a dynamic slice of static binary blobs if
// present in a fork.
func HashSliceOfStaticBytesOnFork[T commonBytesLengths](h *Hasher, blobs []T, maxItems uint64, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashSliceOfStaticBytes(h, blobs, maxItems)
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

// HashSliceOfDynamicBytesOnFork hashes a dynamic slice of dynamic binary blobs
// if present in a fork.
func HashSliceOfDynamicBytesOnFork(h *Hasher, blobs [][]byte, maxItems uint64, maxSize uint64, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashSliceOfDynamicBytes(h, blobs, maxItems, maxSize)
}

// HashSliceOfStaticObjects hashes a dynamic slice of static ssz objects.
func HashSliceOfStaticObjects[T StaticObject](h *Hasher, objects []T, maxItems uint64) {
	h.descendMixinLayer()
	defer h.ascendMixinLayer(uint64(len(objects)), maxItems)

	// If threading is disabled, or hashing nothing, do it sequentially
	if !h.threads || len(objects) == 0 || len(objects)*int(Size(objects[0], h.codec.fork)) < concurrencyThreshold {
		for _, obj := range objects {
			h.descendLayer()
			obj.DefineSSZ(h.codec)
			h.ascendLayer(0)
		}
		return
	}
	// Split the slice into equal chunks and hash the objects concurrently. The
	// splits will in theory be objects // threads. In practice, we need powers
	// of 2, otherwise child hashers wouldn't be able to collapse their tasks
	// into a single sub-root. Going for the biggest power of two that can be
	// served by exactly N threads is a problem, because we can end up with N/2-1
	// threads idling at worse. To avoid starvation, we're splitting across a
	// higher thead count than cores.
	var workers errgroup.Group
	workers.SetLimit(runtime.NumCPU())

	var (
		splits  = min(4*runtime.NumCPU(), len(objects))
		subtask = max(1<<bitops.Len(uint(len(objects)/splits)), 1)

		resultChunks = make([][32]byte, (len(objects)+subtask-1)/subtask)
		resultDepths = make([]int, (len(objects)+subtask-1)/subtask)
	)
	for i := 0; i < len(resultChunks); i++ {
		worker := i // Take care, closure

		workers.Go(func() error {
			codec := hasherPool.Get().(*Codec)
			defer hasherPool.Put(codec)
			defer codec.has.Reset()
			codec.has.threads = true

			for i := worker * subtask; i < (worker+1)*subtask && i < len(objects); i++ {
				codec.has.descendLayer()
				objects[i].DefineSSZ(codec)
				codec.has.ascendLayer(0)
			}
			codec.has.balanceLayer()

			resultChunks[worker] = codec.has.chunks[0]
			resultDepths[worker] = codec.has.groups[0].depth
			return nil
		})
	}
	// Wait for all the hashers to finish and aggregate the results
	workers.Wait()
	for i := 0; i < len(resultChunks); i++ {
		h.insertChunk(resultChunks[i], resultDepths[i])
	}
}

// HashSliceOfStaticObjectsOnFork hashes a dynamic slice of static ssz objects
// if present in a fork.
func HashSliceOfStaticObjectsOnFork[T StaticObject](h *Hasher, objects []T, maxItems uint64, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashSliceOfStaticObjects(h, objects, maxItems)
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

// HashSliceOfDynamicObjectsOnFork hashes a dynamic slice of dynamic ssz objects
// if present in a fork.
func HashSliceOfDynamicObjectsOnFork[T DynamicObject](h *Hasher, objects []T, maxItems uint64, filter ForkFilter) {
	// If the field is not active in the current fork, early return
	if h.codec.fork < filter.Added || (filter.Removed > ForkUnknown && h.codec.fork >= filter.Removed) {
		return
	}
	// Otherwise fall back to the standard hasher
	HashSliceOfDynamicObjects(h, objects, maxItems)
}

// hashBytes either appends the blob to the hasher's scratch space if it's small
// enough to fit into a single chunk, or chunks it up and merkleizes it first.
func (h *Hasher) hashBytes(blob []byte) {
	// If the blob is small, accumulate as a single chunk
	if len(blob) <= 32 {
		var buffer [32]byte
		copy(buffer[:], blob)
		h.insertChunk(buffer, 0)
		return
	}
	// Otherwise hash it as its own tree
	h.descendLayer()
	h.insertBlobChunks(blob)
	h.ascendLayer(0)
}

// hashBytesEmpty is analogous to hashBytes, but where the input is all zeroes,
// so it's passed by length, not by content. This allows hashing zero pointers
// without allocating them first.
func (h *Hasher) hashBytesEmpty(size int) {
	// If the blob is small, accumulate as a single chunk
	if size <= 32 {
		h.insertChunk(hasherZeroChunk, 0)
		return
	}
	// Otherwise hash it as its own tree
	h.descendLayer()
	h.insertBlobChunksEmpty(size)
	h.ascendLayer(0)
}

// insertChunk adds a chunk to the accumulators, collapsing matching pairs.
func (h *Hasher) insertChunk(chunk [32]byte, depth int) {
	// Insert the chunk into the accumulator
	h.chunks = append(h.chunks, chunk)

	// If the depth tracker is at the leaf level, bump the leaf count
	groups := len(h.groups)
	if groups > 0 && h.groups[groups-1].layer == h.layer && h.groups[groups-1].depth == depth {
		h.groups[groups-1].chunks++
	} else {
		// New leaf group, create it and early return. Nothing to hash with only
		// one leaf in our chunk list.
		h.groups = append(h.groups, groupStats{
			layer:  h.layer,
			depth:  depth,
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
		group.chunks >>= 1

		// The last group tracker we've just hashed needs to be either updated to
		// the new level count, or merged into the previous one if they share all
		// the layer/depth params.
		if groups > 1 {
			prev := h.groups[groups-2]
			if prev.layer == group.layer && prev.depth == group.depth {
				// Two groups can be merged, will trigger a new collapse round
				prev.chunks += group.chunks
				group = prev

				groups--
				continue
			}
		}
		// Either have a single group, or the previous is from a different layer
		// or depth level, update the tail and return
		h.groups = h.groups[:groups]
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
		h.insertChunk(buffer, 0)
		blob = blob[32:]
	}
	if len(blob) > 0 {
		buffer = [32]byte{}
		copy(buffer[:], blob)
		h.insertChunk(buffer, 0)
	}
}

// insertBlobChunksEmpty is analogous to insertBlobChunks, but where the input
// is all zeroes, so it's passed by length, not by content. This allows hashing
// zero pointers without allocating them first.
func (h *Hasher) insertBlobChunksEmpty(size int) {
	for size > 0 { // will insert a full chunk for the last segment
		h.insertChunk(hasherZeroChunk, 0)
		size -= 32
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
	// Before even considering extending the layer to capacity, balance any
	// partial sub-tries to their completion.
	h.balanceLayer()

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

	h.insertChunk(root, 0)
}

// balanceLayer can be used to take a partial hashing result of an unbalanced
// trie and append enough empty chunks (virtually) at the end to collapse it
// down to a single root.
func (h *Hasher) balanceLayer() {
	// If the layer is incomplete, append in zero chunks. First up, before even
	// caring about maximum length, we must balance the tree (i.e. reduce it to
	// a single root hash).
	for {
		groups := len(h.groups)

		// If the last layer was reduced to one root, we've balanced the tree
		group := h.groups[groups-1]
		if group.chunks == 1 {
			if groups == 1 || h.groups[groups-2].layer != group.layer {
				return
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
		h.chunks = h.chunks[:chunks-int(group.chunks)>>1]

		group.depth++
		group.chunks >>= 1

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
}

// ascendMixinLayer is similar to ascendLayer, but actually ascends one for the
// data content, and then mixes in the provided length and ascends once more.
func (h *Hasher) ascendMixinLayer(size uint64, chunks uint64) {
	// If no items have been added, there's nothing to ascend out of. Fix that
	// corner-case here.
	var buffer [32]byte
	if size == 0 {
		h.insertChunk(buffer, 0)
	}
	h.ascendLayer(chunks) // data content

	binary.LittleEndian.PutUint64(buffer[:8], size)
	h.insertChunk(buffer, 0)

	h.ascendLayer(0) // length mixin
}

// Reset resets the Hasher obj
func (h *Hasher) Reset() {
	h.chunks = h.chunks[:0]
	h.groups = h.groups[:0]
	h.threads = false
}
