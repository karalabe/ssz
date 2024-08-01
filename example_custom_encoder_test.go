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

func ExampleCustomEncoder() {
	rawCdc := &ssz.Codec{}
	rawCdc.SetEncoder(new(ssz.Encoder))
	cdc := CustomCodec{Codec: rawCdc}
	hash := ssz.HashWithCodecSequential(cdc, new(Withdrawal))

	fmt.Printf("hash: %#x\n", hash)
	// Output
	// hash: 0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71
}
