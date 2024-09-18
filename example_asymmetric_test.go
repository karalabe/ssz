// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz_test

import (
	"fmt"

	"github.com/karalabe/ssz"
)

type WithdrawalAsym struct {
	Index     uint64  `ssz-size:"8"`
	Validator uint64  `ssz-size:"8"`
	Address   Address `ssz-size:"20"`
	Amount    uint64  `ssz-size:"8"`
}

func (w *WithdrawalAsym) SizeSSZ(siz *ssz.Sizer) uint32 { return 44 }

func (w *WithdrawalAsym) DefineSSZ(codec *ssz.Codec) {
	codec.DefineEncoder(func(enc *ssz.Encoder) {
		ssz.EncodeUint64(enc, w.Index)         // Field (0) - Index          -  8 bytes
		ssz.EncodeUint64(enc, w.Validator)     // Field (1) - ValidatorIndex -  8 bytes
		ssz.EncodeStaticBytes(enc, &w.Address) // Field (2) - Address        - 20 bytes
		ssz.EncodeUint64(enc, w.Amount)        // Field (3) - Amount         -  8 bytes
	})
	codec.DefineDecoder(func(dec *ssz.Decoder) {
		ssz.DecodeUint64(dec, &w.Index)        // Field (0) - Index          -  8 bytes
		ssz.DecodeUint64(dec, &w.Validator)    // Field (1) - ValidatorIndex -  8 bytes
		ssz.DecodeStaticBytes(dec, &w.Address) // Field (2) - Address        - 20 bytes
		ssz.DecodeUint64(dec, &w.Amount)       // Field (3) - Amount         -  8 bytes
	})
	codec.DefineHasher(func(has *ssz.Hasher) {
		ssz.HashUint64(has, w.Index)         // Field (0) - Index          -  8 bytes
		ssz.HashUint64(has, w.Validator)     // Field (1) - ValidatorIndex -  8 bytes
		ssz.HashStaticBytes(has, &w.Address) // Field (2) - Address        - 20 bytes
		ssz.HashUint64(has, w.Amount)        // Field (3) - Amount         -  8 bytes
	})
}

func ExampleEncodeAsymmetricObject() {
	blob := make([]byte, ssz.Size((*WithdrawalAsym)(nil)))
	if err := ssz.EncodeToBytes(blob, new(WithdrawalAsym)); err != nil {
		panic(err)
	}
	hash := ssz.HashSequential(new(WithdrawalAsym))

	fmt.Printf("ssz: %#x\nhash: %#x\n", blob, hash)
	// Output:
	// ssz: 0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
	// hash: 0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71
}
