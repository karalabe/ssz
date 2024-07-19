package ssz

// import (
// 	"unsafe"

// 	"github.com/holiman/uint256"
// )

// type Treerer struct {
// 	codec *Codec   // Self-referencing to pass DefineSSZ calls through (API trick)
// 	buf   [32]byte // Buffer for temporary operations

// 	// Root node of the Merkle tree
// 	root *Node

// 	// Current node being processed
// 	currentNode *Node

// 	// Stack to keep track of nodes during tree construction
// 	nodeStack []*Node

// 	// Depth of the current node in the tree
// 	depth uint32

// 	// Maximum depth allowed for the tree
// 	maxDepth uint32

// 	// Error encountered during tree construction
// 	err error
// }

// type Node struct {
// 	Value [32]byte

// 	Left  *Node
// 	Right *Node
// }

// // CreateNode creates a new Node with the given value and optional left and right child nodes.
// func CreateNode(value [32]byte, left, right *Node) *Node {
// 	return &Node{
// 		Value: value,
// 		Left:  left,
// 		Right: right,
// 	}
// }

// func TreeifyBool(value bool) *Node {
// 	var b [32]byte
// 	if value {
// 		b[0] = 1
// 	}
// 	return CreateNode(b, nil, nil)
// }

// func TreeifyUint64(tre *Treerer, value uint64) *Node {
// 	EncodeUint64(tre.codec.enc, value)
// 	return CreateNode(tre.codec.enc.buf, nil, nil)
// }

// func TreeifyUint256(tre *Treerer, value *uint256.Int) *Node {
// 	var b [32]byte
// 	if value != nil {
// 		EncodeUint256(tre.codec.enc, value)
// 	} else {
// 		copy(b[:], uint256Zero)
// 	}
// 	return CreateNode(b, nil, nil)
// }

// // TreeifyStaticBytes creates a new Node with the Merkle root hash of the given static bytes.
// func TreeifyStaticBytes[T commonBytesLengths](tre *Treerer, blob *T) *Node {
// 	// Create a hasher to process the static bytes
// 	hasher := tre.codec.has

// 	// Hash the static bytes
// 	hasher.hashBytes(unsafe.Slice(&(*blob)[0], len(*blob)))

// 	// Balance the layer to ensure we have a single root
// 	hasher.balanceLayer()

// 	// Extract the root hash
// 	root := hasher.chunks[len(hasher.chunks)-1]

// 	// Reset the hasher for future use
// 	hasher.Reset()

// 	// Create and return a new Node with the root hash
// 	return CreateNode(root, nil, nil)
// }
