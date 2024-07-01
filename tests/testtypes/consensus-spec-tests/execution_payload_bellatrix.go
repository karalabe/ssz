// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import (
	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
)

type ExecutionPayloadBellatrix struct {
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
}

func (e *ExecutionPayloadBellatrix) StaticSSZ() bool { return false }
func (e *ExecutionPayloadBellatrix) SizeSSZ() uint32 {
	// Start out with the static size
	size := uint32(508)

	// Append all the dynamic sizes
	size += ssz.SizeDynamicBlob(e.ExtraData)     // Field (10) - ExtraData    - max 32 bytes (not enforced)
	size += ssz.SizeDynamicBlobs(e.Transactions) // Field (13) - Transactions - max 1048576 items, 1073741824 bytes each (not enforced)

	return size
}

func (e *ExecutionPayloadBellatrix) EncodeSSZ(enc *ssz.Encoder) {
	// Initialize a dynamic encoder with the given starting offsets
	defer enc.OffsetDynamics(508)()

	// If we're serializing Bellatrix or later
	ssz.EncodeBinary(enc, e.ParentHash[:])      // Field  ( 0) - ParentHash    -  32 bytes
	ssz.EncodeBinary(enc, e.FeeRecipient[:])    // Field  ( 1) - FeeRecipient  -  20 bytes
	ssz.EncodeBinary(enc, e.StateRoot[:])       // Field  ( 2) - StateRoot     -  32 bytes
	ssz.EncodeBinary(enc, e.ReceiptsRoot[:])    // Field  ( 3) - ReceiptsRoot  -  32 bytes
	ssz.EncodeBinary(enc, e.LogsBloom[:])       // Field  ( 4) - LogsBloom     - 256 bytes
	ssz.EncodeBinary(enc, e.PrevRandao[:])      // Field  ( 5) - PrevRandao    -  32 bytes
	ssz.EncodeUint64(enc, e.BlockNumber)        // Field  ( 6) - BlockNumber   -   8 bytes
	ssz.EncodeUint64(enc, e.GasLimit)           // Field  ( 7) - GasLimit      -   8 bytes
	ssz.EncodeUint64(enc, e.GasUsed)            // Field  ( 8) - GasUsed       -   8 bytes
	ssz.EncodeUint64(enc, e.Timestamp)          // Field  ( 9) - Timestamp     -   8 bytes
	ssz.EncodeDynamicBlob(enc, e.ExtraData)     // Offset (10) - ExtraData     -   4 bytes + later max 32 bytes (not enforced)
	ssz.EncodeUint256(enc, e.BaseFeePerGas)     // Field  (11) - BaseFeePerGas -  32 bytes
	ssz.EncodeBinary(enc, e.BlockHash[:])       // Field  (12) - BlockHash     -  32 bytes
	ssz.EncodeDynamicBlobs(enc, e.Transactions) // Offset (13) - Transactions  -   4 bytes + later max 1048576 items, 1073741824 bytes each (not enforced)
}

func (e *ExecutionPayloadBellatrix) DecodeSSZ(dec *ssz.Decoder) {
	// Initialize a dynamic decoder with the given starting offsets
	defer dec.OffsetDynamics(508)()

	// If we're parsing Bellatrix or later
	ssz.DecodeBinary(dec, e.ParentHash[:])                                 // Field  ( 0) - ParentHash    -  32 bytes
	ssz.DecodeBinary(dec, e.FeeRecipient[:])                               // Field  ( 1) - FeeRecipient  -  20 bytes
	ssz.DecodeBinary(dec, e.StateRoot[:])                                  // Field  ( 2) - StateRoot     -  32 bytes
	ssz.DecodeBinary(dec, e.ReceiptsRoot[:])                               // Field  ( 3) - ReceiptsRoot  -  32 bytes
	ssz.DecodeBinary(dec, e.LogsBloom[:])                                  // Field  ( 4) - LogsBloom     - 256 bytes
	ssz.DecodeBinary(dec, e.PrevRandao[:])                                 // Field  ( 5) - PrevRandao    -  32 bytes
	ssz.DecodeUint64(dec, &e.BlockNumber)                                  // Field  ( 6) - BlockNumber   -   8 bytes
	ssz.DecodeUint64(dec, &e.GasLimit)                                     // Field  ( 7) - GasLimit      -   8 bytes
	ssz.DecodeUint64(dec, &e.GasUsed)                                      // Field  ( 8) - GasUsed       -   8 bytes
	ssz.DecodeUint64(dec, &e.Timestamp)                                    // Field  ( 9) - Timestamp     -   8 bytes
	ssz.DecodeDynamicBlob(dec, &e.ExtraData, 32)                           // Offset (10) - ExtraData     -   4 bytes
	ssz.DecodeUint256(dec, &e.BaseFeePerGas)                               // Field  (11) - BaseFeePerGas -  32 bytes
	ssz.DecodeBinary(dec, e.BlockHash[:])                                  // Field  (12) - BlockHash     -  32 bytes
	ssz.DecodeDynamicBlobs(dec, &e.Transactions, 1_048_576, 1_073_741_824) // Offset (13) - Transactions  -   4 bytes
}
