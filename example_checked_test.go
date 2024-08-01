// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz_test

import (
	"fmt"

	"github.com/karalabe/ssz"
)

type WithdrawalChecked struct {
	Index     uint64 `ssz-size:"8"`
	Validator uint64 `ssz-size:"8"`
	Address   []byte `ssz-size:"20"`
	Amount    uint64 `ssz-size:"8"`
}

func (w *WithdrawalChecked) SizeSSZ() uint32 { return 44 }

func (w *WithdrawalChecked) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &w.Index)                   // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)               // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineCheckedStaticBytes(codec, &w.Address, 20) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)                  // Field (3) - Amount         -  8 bytes
}

func ExampleDecodeCheckedObject() {
	blob := make([]byte, 44)

	obj := new(WithdrawalChecked)
	if err := ssz.DecodeFromBytes(blob, obj); err != nil {
		panic(err)
	}
	fmt.Printf("obj: %#x\n", obj)
	// Output:
	// obj: &{0x0 0x0 0x0000000000000000000000000000000000000000 0x0}
}
