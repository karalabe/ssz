package ssz

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"unsafe"

	"github.com/holiman/uint256"
	"github.com/minio/sha256-simd"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/gohashtree"
)

// Some helpers to avoid occasional allocations
var (
	hasherBoolFalse = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	hasherBoolTrue  = []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

// Hasher is an SSZ Merkle Hash Root computer.
type Hasher struct {
	scratch []byte // Scratch space for not-yet-hashed writes

	codec *Codec   // Self-referencing to pass DefineSSZ calls through (API trick)
	buf   [32]byte // Integer conversion buffer
}

// HashBool hashes a boolean.
func HashBool[T ~bool](h *Hasher, v T) {
	h.PutBool((bool)(v))
}

// HashUint64 hashes a uint64.
func HashUint64[T ~uint64](h *Hasher, n T) {
	binary.LittleEndian.PutUint64(h.buf[:8], (uint64)(n))
	h.AppendBytes32(h.buf[:8])
}

// HashUint256 hashes a uint256.
//
// Note, a nil pointer is hashed as zero.
func HashUint256(h *Hasher, n *uint256.Int) {
	if n != nil {
		n.MarshalSSZInto(h.buf[:32])
		h.PutBytes(h.buf[:32])
	} else {
		h.PutBytes(uint256Zero)
	}
}

// HashStaticBytes hashes a static binary blob.
//
// The blob is passed by pointer to avoid high stack copy costs and a potential
// escape to the heap.
func HashStaticBytes[T commonBytesLengths](h *Hasher, blob *T) {
	// The code below should have used `blob[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	h.PutBytes(unsafe.Slice(&(*blob)[0], len(*blob)))
}

// HashCheckedStaticBytes hashes a static binary blob.
func HashCheckedStaticBytes(h *Hasher, blob []byte) {
	h.PutBytes(blob)
}

// HashDynamicBytes hashes a dynamic binary blob.
func HashDynamicBytes(h *Hasher, blob []byte, maxSize uint64) {
	idx := h.Index()
	h.Append(blob)
	h.MerkleizeWithMixin(idx, uint64(len(blob)), (maxSize+31)/32)
}

// HashStaticObject hashes a static ssz object.
func HashStaticObject(h *Hasher, obj StaticObject) {
	idx := h.Index()
	obj.DefineSSZ(h.codec)
	h.Merkleize(idx)
}

// HashDynamicObject hashes a dynamic ssz object.
func HashDynamicObject(h *Hasher, obj DynamicObject) {
	idx := h.Index()
	obj.DefineSSZ(h.codec)
	h.Merkleize(idx)
}

// HashArrayOfBits hashes a static array of (packed) bits.
func HashArrayOfBits[T commonBitsLengths](h *Hasher, bits *T) {
	// The code below should have used `*bits[:]`, alas Go's generics compiler
	// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
	h.PutBytes(unsafe.Slice(&(*bits)[0], len(*bits)))
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

	idx := h.Index()
	for _, n := range nums {
		binary.LittleEndian.PutUint64(h.buf[:8], n)
		h.Append(h.buf[:8])
	}
	h.Merkleize(idx)
}

// HashSliceOfUint64s hashes a dynamic slice of uint64s.
func HashSliceOfUint64s[T ~uint64](h *Hasher, ns []T, maxItems uint64) {
	idx := h.Index()
	for _, n := range ns {
		binary.LittleEndian.PutUint64(h.buf[:8], (uint64)(n))
		h.Append(h.buf[:8])
	}
	h.FillUpTo32()
	h.MerkleizeWithMixin(idx, uint64(len(ns)), (maxItems*8+31)/32)
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
	idx := h.Index()
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.PutBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.Merkleize(idx)
}

// HashCheckedArrayOfStaticBytes hashes a static array of static binary blobs.
func HashCheckedArrayOfStaticBytes[T commonBytesLengths](h *Hasher, blobs []T) {
	idx := h.Index()
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.PutBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.Merkleize(idx)
}

// HashSliceOfStaticBytes hashes a dynamic slice of static binary blobs.
func HashSliceOfStaticBytes[T commonBytesLengths](h *Hasher, blobs []T, maxItems uint64) {
	idx := h.Index()
	for i := 0; i < len(blobs); i++ {
		// The code below should have used `blobs[i][:]`, alas Go's generics compiler
		// is missing that (i.e. a bug): https://github.com/golang/go/issues/51740
		h.PutBytes(unsafe.Slice(&blobs[i][0], len(blobs[i])))
	}
	h.MerkleizeWithMixin(idx, uint64(len(blobs)), maxItems)
}

// HashSliceOfDynamicBytes hashes a dynamic slice of dynamic binary blobs.
func HashSliceOfDynamicBytes(h *Hasher, blobs [][]byte, maxItems uint64, maxSize uint64) {
	idx := h.Index()
	for _, blob := range blobs {
		idx := h.Index()
		h.AppendBytes32(blob)
		h.MerkleizeWithMixin(idx, uint64(len(blob)), (maxSize+31)/32)
	}
	h.MerkleizeWithMixin(idx, uint64(len(blobs)), maxItems)
}

// HashSliceOfStaticObjects hashes a dynamic slice of static ssz objects.
func HashSliceOfStaticObjects[T StaticObject](h *Hasher, objects []T, maxItems uint64) {
	idx := h.Index()
	for _, obj := range objects {
		idx := h.Index()
		obj.DefineSSZ(h.codec)
		h.Merkleize(idx)
	}
	h.MerkleizeWithMixin(idx, uint64(len(objects)), maxItems)
}

// HashSliceOfDynamicObjects hashes a dynamic slice of dynamic ssz objects.
func HashSliceOfDynamicObjects[T DynamicObject](h *Hasher, objects []T, maxItems uint64) {
	idx := h.Index()
	for _, obj := range objects {
		idx := h.Index()
		obj.DefineSSZ(h.codec)
		h.Merkleize(idx)
	}
	h.MerkleizeWithMixin(idx, uint64(len(objects)), maxItems)
}

var zeroHashes [65][32]byte
var zeroHashLevels map[string]int
var trueBytes, falseBytes []byte

func init() {
	falseBytes = make([]byte, 32)
	trueBytes = make([]byte, 32)
	trueBytes[0] = 1
	zeroHashLevels = make(map[string]int)
	zeroHashLevels[string(falseBytes)] = 0

	tmp := [64]byte{}
	for i := 0; i < 64; i++ {
		copy(tmp[:32], zeroHashes[i][:])
		copy(tmp[32:], zeroHashes[i][:])
		zeroHashes[i+1] = sha256.Sum256(tmp[:])
		zeroHashLevels[string(zeroHashes[i+1][:])] = i + 1
	}
}

var zeroBytes = make([]byte, 32)

// Reset resets the Hasher obj
func (h *Hasher) Reset() {
	h.scratch = h.scratch[:0]
}

func (h *Hasher) AppendBytes32(b []byte) {
	h.scratch = append(h.scratch, b...)
	if rest := len(b) % 32; rest != 0 {
		// pad zero bytes to the left
		h.scratch = append(h.scratch, zeroBytes[:32-rest]...)
	}
}

func (h *Hasher) FillUpTo32() {
	// pad zero bytes to the left
	if rest := len(h.scratch) % 32; rest != 0 {
		h.scratch = append(h.scratch, zeroBytes[:32-rest]...)
	}
}

func (h *Hasher) Append(i []byte) {
	h.scratch = append(h.scratch, i...)
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
	indx := h.Index()
	h.AppendBytes32(tmp)
	h.MerkleizeWithMixin(indx, size, (maxSize+255)/256)
}

// PutBool appends a boolean
func (h *Hasher) PutBool(b bool) {
	if b {
		h.scratch = append(h.scratch, trueBytes...)
	} else {
		h.scratch = append(h.scratch, falseBytes...)
	}
}

// PutBytes appends bytes
func (h *Hasher) PutBytes(b []byte) {
	if len(b) <= 32 {
		h.AppendBytes32(b)
		return
	}

	// if the bytes are longer than 32 we have to
	// merkleize the content
	indx := h.Index()
	h.AppendBytes32(b)
	h.Merkleize(indx)
}

// Index marks the current buffer index
func (h *Hasher) Index() int {
	return len(h.scratch)
}

// Merkleize is used to merkleize the last group of the hasher
func (h *Hasher) Merkleize(indx int) {
	// merkleizeImpl will expand the `input` by 32 bytes if some hashing depth
	// hits an odd chunk length. But if we're at the end of `h.scratch` already,
	// appending to `input` will allocate a new buffer, *not* expand `h.scratch`,
	// so the next invocation will realloc, over and over and over. We can pre-
	// emptively cater for that by ensuring that an extra 32 bytes is always
	// available.
	if len(h.scratch) == cap(h.scratch) {
		h.scratch = append(h.scratch, zeroBytes...)
		h.scratch = h.scratch[:len(h.scratch)-len(zeroBytes)]
	}
	input := h.scratch[indx:]

	// merkleize the input
	input = h.merkleizeImpl(input[:0], input, 0)
	h.scratch = append(h.scratch[:indx], input...)
}

// MerkleizeWithMixin is used to merkleize the last group of the hasher
func (h *Hasher) MerkleizeWithMixin(indx int, num, limit uint64) {
	h.FillUpTo32()
	input := h.scratch[indx:]

	// merkleize the input
	input = h.merkleizeImpl(input[:0], input, limit)

	// mixin with the size
	output := h.buf[:32]
	for indx := range output {
		output[indx] = 0
	}
	binary.LittleEndian.PutUint64(output[:8], num)
	input = append(input, output...)

	// input is of the form [<input><size>] of 64 bytes
	gohashtree.HashByteSlice(input, input)
	h.scratch = append(h.scratch[:indx], input[:32]...)
}

func (h *Hasher) Hash() []byte {
	return h.scratch[len(h.scratch)-32:]
}

// HashRoot creates the hash final hash root
func (h *Hasher) HashRoot() (res [32]byte, err error) {
	if len(h.scratch) != 32 {
		err = fmt.Errorf("expected 32 byte size")
		return
	}
	copy(res[:], h.scratch)
	return
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
		return append(dst, zeroBytes...)
	}
	if limit == 1 {
		if count == 1 {
			return append(dst, input[:32]...)
		}
		return append(dst, zeroBytes...)
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
