// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"crypto/sha256"
	"encoding/binary"
	bitops "math/bits"
	"runtime"
	"unsafe"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/gohashtree"
	"golang.org/x/sync/errgroup"
)

// concurrencyThreshold is the data size above which a new sub-hasher is spun up
// for each dynamic field instead of hashing sequentially.
const concurrencyThreshold = 4096

// Some helpers to avoid occasional allocations
var (
	hasherBoolFalse = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherBoolTrue  = []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherUint64Pad = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherZeroChunk = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

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
	threads bool   // Whether threaded hashing is allowed or not
	scratch []byte // Scratch space for not-yet-hashed writes

	workers errgroup.Group // Concurrent hashers for large fields
	barrier chan struct{}  // Channel blocking the workers from writing their results (avoids scratch space race)

	codec  *Codec   // Self-referencing to pass DefineSSZ calls through (API trick)
	buf    [32]byte // Integer conversion buffer
	bitbuf []byte   // Bitlist conversion buffer (
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
	// If threading is disabled or the total size to hash is small, do it sequentially
	if !h.threads || len(blob) < concurrencyThreshold {
		// Only hashing bytes, operate in the current hasher context
		pos := len(h.scratch)
		h.scratch = append(h.scratch, blob...)
		h.merkleizeWithMixin(pos, uint64(len(blob)), (maxSize+31)/32)
		return
	}
	// Considerable size, hash the item concurrently
	h.merkleizeConcurrent(func(ch *Hasher) {
		ch.scratch = append(ch.scratch, blob...)
		ch.merkleizeWithMixin(0, uint64(len(blob)), (maxSize+31)/32)
	})
}

// HashStaticObject hashes a static ssz object.
func HashStaticObject(h *Hasher, obj StaticObject) {
	// If threading is disabled, no need for a fresh context
	if !h.threads {
		pos := len(h.scratch)
		obj.DefineSSZ(h.codec)
		h.merkleize(pos, 0)
		return
	}
	// If the total size to hash is small, do it sequentially
	if Size(obj) < concurrencyThreshold {
		h.merkleizeSequential(func(sh *Hasher) {
			obj.DefineSSZ(sh.codec)
			sh.merkleize(0, 0)
		})
		return
	}
	// Considerable size, hash the item concurrently
	h.merkleizeConcurrent(func(ch *Hasher) {
		obj.DefineSSZ(ch.codec)
		ch.merkleize(0, 0)
	})
}

// HashDynamicObject hashes a dynamic ssz object.
func HashDynamicObject(h *Hasher, obj DynamicObject) {
	// If threading is disabled, no need for a fresh context
	if !h.threads {
		pos := len(h.scratch)
		obj.DefineSSZ(h.codec)
		h.merkleize(pos, 0)
		return
	}
	// If the total size to hash is small, do it sequentially
	if Size(obj) < concurrencyThreshold {
		// Use a fresh hasher to avoid mixing child and sibling threads
		h.merkleizeSequential(func(sh *Hasher) {
			obj.DefineSSZ(sh.codec)
			sh.merkleize(0, 0)
		})
		return
	}
	// Considerable size, hash the item concurrently
	h.merkleizeConcurrent(func(ch *Hasher) {
		obj.DefineSSZ(ch.codec)
		ch.merkleize(0, 0)
	})
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
	pos := len(h.scratch)
	h.appendBytesChunks(h.bitbuf)
	h.merkleizeWithMixin(pos, size, (maxBits+255)/256)
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

	// If the total size to hash is small, do it sequentially
	if len(nums)*8 < concurrencyThreshold {
		// Only hashing uints, operate in the current hasher context
		pos := len(h.scratch)
		for _, n := range nums {
			binary.LittleEndian.PutUint64(h.buf[:8], n)
			h.scratch = append(h.scratch, h.buf[:8]...)
		}
		h.merkleize(pos, 0)
		return
	}
	// Considerable size, hash the item concurrently
	h.merkleizeConcurrent(func(ch *Hasher) {
		for _, n := range nums {
			binary.LittleEndian.PutUint64(ch.buf[:8], n)
			ch.scratch = append(ch.scratch, ch.buf[:8]...)
		}
		ch.merkleize(0, 0)
	})
}

// HashSliceOfUint64s hashes a dynamic slice of uint64s.
func HashSliceOfUint64s[T ~uint64](h *Hasher, ns []T, maxItems uint64) {
	// If threading is disabled or the total size to hash is small, do it sequentially
	if !h.threads || len(ns)*8 < concurrencyThreshold {
		// Only hashing uints, operate in the current hasher context
		pos := len(h.scratch)
		for _, n := range ns {
			binary.LittleEndian.PutUint64(h.buf[:8], (uint64)(n))
			h.scratch = append(h.scratch, h.buf[:8]...)
		}
		h.merkleizeWithMixin(pos, uint64(len(ns)), (maxItems*8+31)/32)
		return
	}
	// Considerable size, hash the item concurrently
	h.merkleizeConcurrent(func(ch *Hasher) {
		for _, n := range ns {
			binary.LittleEndian.PutUint64(ch.buf[:8], (uint64)(n))
			ch.scratch = append(ch.scratch, ch.buf[:8]...)
		}
		ch.merkleizeWithMixin(0, uint64(len(ns)), (maxItems*8+31)/32)
	})
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
	h.merkleize(pos, 0)
}

// HashCheckedArrayOfStaticBytes hashes a static array of static binary blobs.
func HashCheckedArrayOfStaticBytes[T commonBytesLengths](h *Hasher, blobs []T) {
	pos := len(h.scratch)
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.hashBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.merkleize(pos, 0)
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
	// If threading is disabled, no need for a fresh context
	if !h.threads {
		pos := len(h.scratch)
		for _, obj := range objects {
			pos := len(h.scratch)
			obj.DefineSSZ(h.codec)
			h.merkleize(pos, 0)
		}
		h.merkleizeWithMixin(pos, uint64(len(objects)), maxItems)
		return
	}
	// If the total size to hash is small, do it sequentially
	if len(objects) == 0 || uint32(len(objects))*Size(objects[0]) < concurrencyThreshold {
		pos := len(h.scratch)
		for _, obj := range objects {
			h.merkleizeSequential(func(sh *Hasher) {
				obj.DefineSSZ(sh.codec)
				sh.merkleize(0, 0)
			})
		}
		h.merkleizeWithMixin(pos, uint64(len(objects)), maxItems)
		return
	}
	// Considerable size, hash the items concurrently
	h.merkleizeConcurrent(func(sliceHasher *Hasher) {
		// Allocate a large enough scratch space for all the object hashes
		for i := 0; i < len(objects); i++ {
			sliceHasher.scratch = append(sliceHasher.scratch, hasherZeroChunk...)
		}
		// Split the slice into equal chunks and hash the objects concurrently
		threads := min(runtime.NumCPU(), len(objects))
		for i := 0; i < threads; i++ {
			var (
				start = len(objects) * i / threads
				end   = len(objects) * (i + 1) / threads
			)
			sliceHasher.workers.Go(func() error {
				for j := start; j < end; j++ {
					// Use a fresh hasher to avoid mixing child and sibling threads
					objHasher := hasherPool.Get().(*Codec)
					objHasher.has.threads = true

					objects[j].DefineSSZ(objHasher)
					objHasher.has.merkleize(0, 0)

					copy(sliceHasher.scratch[j*32:], objHasher.has.scratch[:32])

					objHasher.has.Reset()
					hasherPool.Put(objHasher)
				}
				return nil
			})
		}
		sliceHasher.merkleizeWithMixin(0, uint64(len(objects)), maxItems)
	})
}

// HashSliceOfDynamicObjects hashes a dynamic slice of dynamic ssz objects.
func HashSliceOfDynamicObjects[T DynamicObject](h *Hasher, objects []T, maxItems uint64) {
	// If threading is disabled, no need for a fresh context
	if !h.threads {
		pos := len(h.scratch)
		for _, obj := range objects {
			pos := len(h.scratch)
			obj.DefineSSZ(h.codec)
			h.merkleize(pos, 0)
		}
		h.merkleizeWithMixin(pos, uint64(len(objects)), maxItems)
		return
	}
	// Threading enabled, we need a fresh context
	pos := len(h.scratch)
	for _, obj := range objects {
		h.merkleizeSequential(func(sh *Hasher) {
			obj.DefineSSZ(sh.codec)
			sh.merkleize(0, 0)
		})
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
	h.merkleize(pos, 0)
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

// Reset resets the Hasher obj
func (h *Hasher) Reset() {
	h.scratch = h.scratch[:0]
	h.barrier = nil
	h.threads = false
}

// merkleize hashes everything in the scratch space from the starting position,
// adhering to the requested chunk limit for dynamic data types, or the data
// limit for static ones (chunks == 0).
func (h *Hasher) merkleize(offset int, chunks uint64) {
	// If the offset is 0, we are running the big outer hashing to sum up all
	// the fields of the struct. Make sure any concurrent hasher is done at
	// this point.
	if offset == 0 {
		if h.barrier != nil {
			close(h.barrier)
		}
		h.workers.Wait()
	}
	// If the data size is zero and static hashing (chunks == 0) or singleton
	// chunking (chunk == 1) was requested, return all zeroes.
	size := len(h.scratch) - offset
	if size == 0 && chunks < 2 {
		h.scratch = append(h.scratch[:offset], hasherZeroChunk...)
		return
	}
	// If no chunk limit was specified and the data is not empty, use needed
	// number of chunks. Special case having only one chunk.
	need := uint64((size + 31) / 32)
	if chunks == 0 {
		chunks = need
	}
	if chunks == 1 && need == 1 {
		h.scratch = h.scratch[:offset+32]
		return
	}
	// Many chunks needed, need to recursively compute the hash of the tree
	depth := uint8(bitops.Len64(chunks - 1))
	if size == 0 {
		h.scratch = append(h.scratch[:offset], hasherZeroCache[depth][:]...)
		return
	}
	for i := uint8(0); i < depth; i++ {
		chunks := (len(h.scratch) - offset) >> 5
		if chunks&0x1 == 1 {
			h.scratch = append(h.scratch, hasherZeroCache[i][:]...)
			chunks++
		}
		gohashtree.HashByteSlice(h.scratch[offset:], h.scratch[offset:])
		h.scratch = h.scratch[:offset+(chunks<<4)]
	}
}

// merkleizeWithMixin hashes everything in the scratch space from the starting
// position, also mixing in the size of the dynamic slice of data.
func (h *Hasher) merkleizeWithMixin(pos int, size uint64, chunks uint64) {
	// Fill the scratch space up to a chunk boundary
	if rest := (len(h.scratch) - pos) & 0x1f; rest != 0 {
		h.scratch = append(h.scratch, hasherZeroChunk[:32-rest]...)
	}
	h.merkleize(pos, chunks)

	binary.LittleEndian.PutUint64(h.buf[:8], size)
	h.scratch = append(h.scratch, h.buf[:8]...)
	h.scratch = append(h.scratch, hasherUint64Pad...)

	gohashtree.HashByteSlice(h.scratch[pos:], h.scratch[pos:])
	h.scratch = h.scratch[:pos+32]
}

// merkleizeSequential runs a merkle calculation on the same goroutine, but in
// a nex hashing context.
func (h *Hasher) merkleizeSequential(f func(sh *Hasher)) {
	codec := hasherPool.Get().(*Codec)
	defer hasherPool.Put(codec)
	defer codec.has.Reset()

	f(codec.has)
	h.scratch = append(h.scratch, codec.has.scratch[:32]...)
}

// merkleizeConcurrent runs a merkle calculation on a separate goroutine, and in
// a new haching context.
func (h *Hasher) merkleizeConcurrent(f func(ch *Hasher)) {
	// Make room for the results in the origin scratch space
	position := len(h.scratch)
	h.scratch = append(h.scratch, hasherZeroChunk...)

	// If no concurrent hasher was started until now, initialize the barrier.
	// This will be needed later to avoid the child hasher writing racely into
	// the scratch space while the sequential hasher is expanding it.
	if h.barrier == nil {
		h.barrier = make(chan struct{})
	}
	// Schedule the concurrent hashing
	h.workers.Go(func() error {
		codec := hasherPool.Get().(*Codec)
		defer hasherPool.Put(codec)
		defer codec.has.Reset()

		codec.has.threads = true
		f(codec.has)

		// Hashing done, wait on the barrier before writing to the scratch space
		<-h.barrier
		copy(h.scratch[position:], codec.has.scratch[:32])
		return nil
	})
}
