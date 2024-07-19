// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz_test

import (
	"bytes"
	"fmt"
	"testing"

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

func TestEncodeStaticObject(t *testing.T) {
	out := new(bytes.Buffer)
	if err := ssz.EncodeToStream(out, new(Withdrawal)); err != nil {
		t.Fatalf("Failed to encode Withdrawal: %v", err)
	}
	hash := ssz.HashSequential(new(Withdrawal))
	expectedSSZ := [44]byte{}
	expectedHash := [32]byte{0xdb, 0x56, 0x11, 0x4e, 0x00, 0xfd, 0xd4, 0xc1, 0xf8, 0x5c, 0x89, 0x2b, 0xf3, 0x5a, 0xc9, 0xa8, 0x92, 0x89, 0xaa, 0xec, 0xb1, 0xeb, 0xd0, 0xa9, 0x6c, 0xde, 0x60, 0x6a, 0x74, 0x8b, 0x5d, 0x71}

	if !bytes.Equal(out.Bytes(), expectedSSZ[:]) {
		t.Errorf("Encoded SSZ mismatch.\nGot:  %#x\nWant: %#x", out.Bytes(), expectedSSZ)
	}

	if hash != expectedHash {
		t.Errorf("Hash mismatch.\nGot:  %#x\nWant: %#x", hash, expectedHash)
	}
}

func TestTreeerSymmetricObject(t *testing.T) {
	withdrawal := &Withdrawal{
		Index:     999,
		Validator: 888,
		Address:   Address{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		Amount:    777,
	}

	treeNode := ssz.TreeSequential(withdrawal)
	fmt.Printf("ROOT %#x\n", treeNode.Hash)
}
