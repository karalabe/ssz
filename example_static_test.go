// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz_test

import (
	"bytes"
	"fmt"

	"github.com/karalabe/ssz"
)

type Address [20]byte

type Withdrawal struct {
	Index     uint64  `ssz-size:"8"`
	Validator uint64  `ssz-size:"8"`
	Address   Address `ssz-size:"20"`
	Amount    uint64  `ssz-size:"8"`
}

func (w *Withdrawal) SizeSSZ() uint32 { return 44 }

func (w *Withdrawal) DefineSSZ(codec *ssz.Codec) {
	fmt.Println("CALLING")
	ssz.DefineUint64(codec, &w.Index)        // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)    // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineStaticBytes(codec, &w.Address) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)       // Field (3) - Amount         -  8 bytes
}

func ExampleEncodeStaticObject() {
	out := new(bytes.Buffer)
	if err := ssz.EncodeToStream(out, new(Withdrawal)); err != nil {
		panic(err)
	}
	hash := ssz.HashSequential(new(Withdrawal))

	fmt.Printf("ssz: %#x\nhash: %#x\n", out, hash)
	// Output:
	// ssz: 0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
	// hash: 0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71
}

// func ExampleTreeerSymmetricObject() {
// 	withdrawal := &Withdrawal{
// 		Index:     123,
// 		Validator: 456,
// 		Address:   Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
// 		Amount:    789,
// 	}

// 	treeNode := ssz.TreeSequential(withdrawal)
// 	fmt.Println("ROOT", treeNode)
// 	// fmt.Printf("Root hash: %#x\n", root.Value)
// 	// fmt.Printf("Left child hash: %#x\n", root.Left.Hash)
// 	// fmt.Printf("Right child hash: %#x\n", root.Right.Hash)

// 	// Output:
// 	// Root hash: 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
// 	// Left child hash: 0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678
// 	// Right child hash: 0x9090909090909090909090909090909090909090909090909090909090909090
// }
