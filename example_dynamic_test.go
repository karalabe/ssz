// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz_test

import (
	"fmt"

	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
)

type Hash [32]byte
type LogsBLoom [256]byte

type ExecutionPayload struct {
	ParentHash    Hash          `ssz-size:"32"`
	FeeRecipient  Address       `ssz-size:"20"`
	StateRoot     Hash          `ssz-size:"32"`
	ReceiptsRoot  Hash          `ssz-size:"32"`
	LogsBloom     LogsBLoom     `ssz-size:"256"`
	PrevRandao    Hash          `ssz-size:"32"`
	BlockNumber   uint64        `ssz-size:"8"`
	GasLimit      uint64        `ssz-size:"8"`
	GasUsed       uint64        `ssz-size:"8"`
	Timestamp     uint64        `ssz-size:"8"`
	ExtraData     []byte        `ssz-max:"32"`
	BaseFeePerGas *uint256.Int  `ssz-size:"32"`
	BlockHash     Hash          `ssz-size:"32"`
	Transactions  [][]byte      `ssz-max:"1048576,1073741824"`
	Withdrawals   []*Withdrawal `ssz-max:"16"`
}

func (e *ExecutionPayload) SizeSSZ(fixed bool) uint32 {
	// Start out with the static size
	size := uint32(512)
	if fixed {
		return size
	}
	// Append all the dynamic sizes
	size += ssz.SizeDynamicBytes(e.ExtraData)           // Field (10) - ExtraData    - max 32 bytes (not enforced)
	size += ssz.SizeSliceOfDynamicBytes(e.Transactions) // Field (13) - Transactions - max 1048576 items, 1073741824 bytes each (not enforced)
	size += ssz.SizeSliceOfStaticObjects(e.Withdrawals) // Field (14) - Withdrawals  - max 16 items, 44 bytes each (not enforced)

	return size
}
func (e *ExecutionPayload) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticBytes(codec, e.ParentHash[:])                                   // Field  ( 0) - ParentHash    -  32 bytes
	ssz.DefineStaticBytes(codec, e.FeeRecipient[:])                                 // Field  ( 1) - FeeRecipient  -  20 bytes
	ssz.DefineStaticBytes(codec, e.StateRoot[:])                                    // Field  ( 2) - StateRoot     -  32 bytes
	ssz.DefineStaticBytes(codec, e.ReceiptsRoot[:])                                 // Field  ( 3) - ReceiptsRoot  -  32 bytes
	ssz.DefineStaticBytes(codec, e.LogsBloom[:])                                    // Field  ( 4) - LogsBloom     - 256 bytes
	ssz.DefineStaticBytes(codec, e.PrevRandao[:])                                   // Field  ( 5) - PrevRandao    -  32 bytes
	ssz.DefineUint64(codec, &e.BlockNumber)                                         // Field  ( 6) - BlockNumber   -   8 bytes
	ssz.DefineUint64(codec, &e.GasLimit)                                            // Field  ( 7) - GasLimit      -   8 bytes
	ssz.DefineUint64(codec, &e.GasUsed)                                             // Field  ( 8) - GasUsed       -   8 bytes
	ssz.DefineUint64(codec, &e.Timestamp)                                           // Field  ( 9) - Timestamp     -   8 bytes
	ssz.DefineDynamicBytes(codec, &e.ExtraData, 32)                                 // Offset (10) - ExtraData     -   4 bytes
	ssz.DefineUint256(codec, &e.BaseFeePerGas)                                      // Field  (11) - BaseFeePerGas -  32 bytes
	ssz.DefineStaticBytes(codec, e.BlockHash[:])                                    // Field  (12) - BlockHash     -  32 bytes
	ssz.DefineSliceOfDynamicBytes(codec, &e.Transactions, 1_048_576, 1_073_741_824) // Offset (13) - Transactions  -   4 bytes
	ssz.DefineSliceOfStaticObjects(codec, &e.Withdrawals, 16)                       // Offset (14) - Withdrawals   -   4 bytes
}

func ExampleEncodeDynamicObject() {
	blob, err := ssz.EncodeToBytes(new(ExecutionPayload))
	if err != nil {
		panic(err)
	}
	fmt.Printf("ssz: %#x\n", blob)
	// Output:
	// ssz: 0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000020000
}
