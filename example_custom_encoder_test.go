// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz_test

import (
	"fmt"

	"github.com/karalabe/ssz"
)

type CustomCodec struct {
	*ssz.Codec
}

type FunkyWithdrawal struct {
	Index     uint64 `ssz-size:"8"`
	Validator uint64 `ssz-size:"8"`
	Address   []byte `ssz-size:"20"`
	Amount    uint64 `ssz-size:"8"`
}

func (w *FunkyWithdrawal) SizeSSZ() uint32 { return 44 }

func (w *FunkyWithdrawal) DefineSSZ(codec ssz.CodecI) {
	ssz.DefineUint64(codec, &w.Index)                   // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)               // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineCheckedStaticBytes(codec, &w.Address, 20) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)                  // Field (3) - Amount         -  8 bytes
}

func ExampleCustomEncoder() {
	rawCdc := &ssz.Codec{}
	rawCdc.SetEncoder(new(ssz.Encoder))
	cdc := CustomCodec{Codec: rawCdc}
	hash := ssz.HashWithCodecSequential(cdc, new(Withdrawal))

	fmt.Printf("hash: %#x\n", hash)
	// Output
	// hash: 0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71
}
