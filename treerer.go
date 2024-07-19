package ssz

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"unsafe"

	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
)

// TreeNode represents a node in the Merkle tree
type TreeNode struct {
	Hash   [32]byte
	Left   *TreeNode
	Right  *TreeNode
	IsLeaf bool
}

// Treerer is a Merkle Tree generator.
type Treerer struct {
	threads bool // Whether threaded hashing is allowed or not

	root   *TreeNode   // Root of the Merkle tree
	leaves []*TreeNode // Leaf nodes of the tree

	codec *Codec // Self-referencing to pass DefineSSZ calls through (API trick)
	// bitbuf []byte // Bitlist conversion buffer
}

// NewTreerer creates a new Treerer instance
func NewTreerer(cdc *Codec) *Treerer {
	fmt.Println("Creating new Treerer instance")
	return &Treerer{
		threads: false,
		leaves:  make([]*TreeNode, 0),
		codec:   cdc,
	}
}

// TreeifyBool creates a new leaf node for a boolean value
func TreeifyBool[T ~bool](t *Treerer, value T) {
	fmt.Printf("TreeifyBool: value=%v\n", value)
	var hash [32]byte
	if value {
		hash[0] = 1
	}
	t.insertLeaf(hash)
}

// TreeifyUint64 creates a new leaf node for a uint64 value
func TreeifyUint64[T ~uint64](t *Treerer, value T) {
	fmt.Printf("TreeifyUint64: value=%d\n", value)
	var hash [32]byte
	binary.LittleEndian.PutUint64(hash[:8], uint64(value))
	t.insertLeaf(hash)
}

// TreeifyUint256 creates a new leaf node for a uint256 value
func TreeifyUint256(t *Treerer, value *uint256.Int) {
	fmt.Printf("TreeifyUint256: value=%v\n", value)
	var hash [32]byte
	if value != nil {
		value.MarshalSSZInto(hash[:])
	}
	t.insertLeaf(hash)
}

// TreeifyStaticBytes creates a new leaf node for a byte slice
func TreeifyStaticBytes(t *Treerer, value []byte) {
	panic("not implemented")
}

// TreeifyCheckedStaticBytes creates a new leaf node for a static byte slice
func TreeifyCheckedStaticBytes(t *Treerer, value []byte) {
	panic("not implemented")
}

// TreeifyDynamicBytes creates a new leaf node for a dynamic byte slice
func TreeifyDynamicBytes(t *Treerer, value []byte) {
	panic("not implemented")
}

// TreeifyCheckedDynamicBytes creates a new leaf node for a checked dynamic byte slice
func TreeifyCheckedDynamicBytes(t *Treerer, value []byte) {
	panic("not implemented")
}

// TreeifyStaticObject creates a new leaf node for a static object
func TreeifyStaticObject(t *Treerer, obj StaticObject) {
	panic("not implemented")
}

// TreeifyDynamicObject creates a new leaf node for a dynamic object
func TreeifyDynamicObject(t *Treerer, obj DynamicObject) {
	panic("not implemented")
}

// TreeifyArrayOfBits computes the hash of an array of bits
func TreeifyArrayOfBits(bits []bool) [32]byte {
	panic("not implemented")
}

// TreeifySliceOfBits creates a new leaf node for a bitlist
func TreeifySliceOfBits(h *Hasher, bits bitfield.Bitlist, maxBits uint64) {
	panic("not implemented")
}

// TreeifyArrayOfUint64s creates leaf nodes for an array of uint64 values
func TreeifyArrayOfUint64s[T commonUint64sLengths](t *Treerer, ns *T) {
	nums := unsafe.Slice(&(*ns)[0], len(*ns))
	var buffer [32]byte

	for len(nums) > 0 {
		for i := 0; i < 4 && i < len(nums); i++ {
			binary.LittleEndian.PutUint64(buffer[i*8:], nums[i])
		}
		t.insertLeaf(buffer)
		if len(nums) > 4 {
			nums = nums[4:]
		} else {
			break
		}
	}
}

// TreeifySliceOfUint64s creates leaf nodes for a slice of uint64 values
func TreeifySliceOfUint64s[T ~uint64](t *Treerer, ns []T, maxItems uint64) {
	var buffer [32]byte

	for len(ns) > 0 {
		for i := 0; i < 4 && i < len(ns); i++ {
			binary.LittleEndian.PutUint64(buffer[i*8:], uint64(ns[i]))
		}
		t.insertLeaf(buffer)
		if len(ns) > 4 {
			ns = ns[4:]
		} else {
			break
		}
	}

	// Add length mix-in
	binary.LittleEndian.PutUint64(buffer[:8], uint64(len(ns)))
	for i := 8; i < 32; i++ {
		buffer[i] = 0
	}
	t.insertLeaf(buffer)

	// Pad with zero nodes if necessary
	zeroNode := [32]byte{}
	for uint64(len(t.leaves)) < (maxItems+3)/4+1 {
		t.insertLeaf(zeroNode)
	}
}

// TreeifyArrayOfStaticBytes creates leaf nodes for a static array of static binary blobs.
func TreeifyArrayOfStaticBytes[T commonBytesArrayLengths[U], U commonBytesLengths](t *Treerer, blobs *T) {
	panic("not implemented")
}

// TreeifyUnsafeArrayOfStaticBytes creates leaf nodes for a static array of static binary blobs.
func TreeifyUnsafeArrayOfStaticBytes[T commonBytesLengths](t *Treerer, blobs []T) {
	panic("not implemented")
}

// TreeifyCheckedArrayOfStaticBytes creates leaf nodes for a static array of static binary blobs.
func TreeifyCheckedArrayOfStaticBytes[T commonBytesLengths](t *Treerer, blobs []T) {
	panic("not implemented")
}

// TreeifySliceOfStaticBytes creates leaf nodes for a dynamic slice of static binary blobs.
func TreeifySliceOfStaticBytes[T commonBytesLengths](t *Treerer, blobs []T, maxItems uint64) {
	panic("not implemented")
}

// TreeifySliceOfDynamicBytes creates leaf nodes for a dynamic slice of dynamic binary blobs.
func TreeifySliceOfDynamicBytes(t *Treerer, blobs [][]byte, maxItems uint64, maxSize uint64) {
	panic("not implemented")
}

// TreeifySliceOfStaticObjects creates leaf nodes for a dynamic slice of static ssz objects.
func TreeifySliceOfStaticObjects[T StaticObject](t *Treerer, objects []T, maxItems uint64) {
	panic("not implemented")
}

// GetRoot returns the root of the Merkle tree
func (t *Treerer) GetRoot() *TreeNode {
	t.balanceAndBuildTree()
	return t.root
}

// insertLeaf adds a new leaf node to the tree
func (t *Treerer) insertLeaf(hash [32]byte) {
	fmt.Printf("Inserting leaf\n")
	fmt.Println("Inserting leaf", hash, "len leaves", len(t.leaves))
	leaf := &TreeNode{
		Hash:   hash,
		IsLeaf: true,
	}
	t.leaves = append(t.leaves, leaf)
}

// balanceAndBuildTree balances the tree and builds it from leaves up
func (t *Treerer) balanceAndBuildTree() {
	fmt.Printf("Balancing and building tree: initial leaves=%d\n", len(t.leaves))
	for len(t.leaves) > 1 {
		var nextLevel []*TreeNode
		for i := 0; i < len(t.leaves); i += 2 {
			var left, right *TreeNode
			left = t.leaves[i]
			if i+1 < len(t.leaves) {
				right = t.leaves[i+1]
			} else {
				right = &TreeNode{Hash: hasherZeroCache[0]}
			}
			parent := &TreeNode{
				Left:  left,
				Right: right,
			}
			parent.Hash = sha256.Sum256(append(left.Hash[:], right.Hash[:]...))
			nextLevel = append(nextLevel, parent)
		}
		fmt.Printf("Tree level processed: new level size=%d\n", len(nextLevel))
		t.leaves = nextLevel
	}
	if len(t.leaves) > 0 {
		t.root = t.leaves[0]
		fmt.Println("Tree root set")
	}
}

// Reset resets the Treerer obj
func (t *Treerer) Reset() {
	fmt.Println("Resetting Treerer")
	if t == nil {
		return
	}
	t.root = nil
	t.leaves = t.leaves[:0]
	t.threads = false
}

// GetRoot returns the root hash of the Merkle tree
func (t *Treerer) GetRoot() *TreeNode {
	fmt.Println("Getting root", "len leaves", len(t.leaves))
	t.balanceAndBuildTree()
	return t.root
}

// PrintTree prints the Merkle tree structure
func PrintTree(root *TreeNode) {
	fmt.Println("Printing Merkle Tree:")
	if root == nil {
		fmt.Println("Tree is empty")
		return
	}
	printNode(root, 0)
}

func printNode(node *TreeNode, level int) {
	if node == nil {
		return
	}

	indent := strings.Repeat("  ", level)
	fmt.Printf("%sValue: %x\n", indent, node.Hash)

	if node.Left != nil || node.Right != nil {
		printNode(node.Left, level+1)
		printNode(node.Right, level+1)
	}
}
