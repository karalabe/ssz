// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import (
	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
)

type ExecutionPayloadCapella struct {
	ParentHash    Hash
	FeeRecipient  Address
	StateRoot     Hash
	ReceiptsRoot  Hash
	LogsBloom     LogsBloom
	PrevRandao    Hash
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	ExtraData     []byte
	BaseFeePerGas *uint256.Int
	BlockHash     Hash
	Transactions  [][]byte
	Withdrawals   []*Withdrawal
}

func (e *ExecutionPayloadCapella) SizeSSZ() uint32 {
	size := uint32(512)
	size += ssz.SizeDynamicBytes(e.ExtraData)           // Field (10) - ExtraData    - max 32 bytes (not enforced)
	size += ssz.SizeSliceOfDynamicBytes(e.Transactions) // Field (13) - Transactions - max 1048576 items, 1073741824 bytes each (not enforced)
	size += ssz.SizeSliceOfStaticObjects(e.Withdrawals) // Field (14) - Withdrawals  - max 16 items, 44 bytes each (not enforced)
	return size
}
func (e *ExecutionPayloadCapella) DefineSSZ(codec *ssz.Codec) {
	codec.OffsetDynamics(512)
	defer codec.FinishDynamics()

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
	ssz.DefineSliceOfStaticObjects(codec, &e.Withdrawals, 16)                       // Offset (14) - Withdrawals - 4 bytes
}
