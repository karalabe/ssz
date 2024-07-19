package ssz

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"unsafe"

	"github.com/holiman/uint256"
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

// TreeSequential computes the ssz merkle tree of the object on a single thread.
// This is useful for processing small objects with stable runtime and O(1) GC
// guarantees.
func TreeSequential(obj Object) *TreeNode {
	codec := treePool.Get().(*Codec)
	defer treePool.Put(codec)
	defer codec.tre.Reset()

	obj.DefineSSZ(codec)
	fmt.Println("LEAVES", len(codec.tre.leaves), "leaves", codec.tre.leaves)
	fmt.Println("Printing all leaves:")
	for i, leaf := range codec.tre.leaves {
		fmt.Printf("Leaf %d: %x\n", i, leaf.Hash)
	}
	return codec.tre.GetRoot()
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

// TreeifyStaticBytes creates a new leaf node with the Merkle root hash of the given static bytes
func TreeifyStaticBytes[T commonBytesLengths](tre *Treerer, blob *T) {
	fmt.Printf("TreeifyStaticBytes: blob length=%d\n", len(*blob))
	hasher := tre.codec.has
	hasher.hashBytes(unsafe.Slice(&(*blob)[0], len(*blob)))
	hasher.balanceLayer()
	root := hasher.chunks[len(hasher.chunks)-1]
	hasher.Reset()
	tre.insertLeaf(root)
}

// TreeifyDynamicBytes creates a new leaf node with the Merkle root hash of the given dynamic bytes
func TreeifyDynamicBytes(tre *Treerer, blob []byte, maxSize uint64) {
	fmt.Printf("TreeifyDynamicBytes: blob length=%d, maxSize=%d\n", len(blob), maxSize)
	hasher := tre.codec.has
	hasher.descendMixinLayer()
	hasher.insertBlobChunks(blob)
	hasher.ascendMixinLayer(uint64(len(blob)), (maxSize+31)/32)
	root := hasher.chunks[len(hasher.chunks)-1]
	hasher.Reset()
	tre.insertLeaf(root)
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
