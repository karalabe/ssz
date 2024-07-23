// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import (
	"math/big"
)

//go:generate go run ../../../cmd/sszgen -type WithdrawalVariation -out gen_withdrawal_variation_ssz.go
//go:generate go run ../../../cmd/sszgen -type HistoricalBatchVariation -out gen_historical_batch_variation_ssz.go
//go:generate go run ../../../cmd/sszgen -type ExecutionPayloadVariation -out gen_execution_payload_variation_ssz.go

type WithdrawalVariation struct {
	Index     uint64
	Validator uint64
	Address   []byte `ssz-size:"20"` // Static bytes defined via ssz-size tag
	Amount    uint64
}

type HistoricalBatchVariation struct {
	BlockRoots [8192]Hash
	StateRoots []Hash `ssz-size:"8192"` // Static array defined via ssz-size tag
}

type ExecutionPayloadVariation struct {
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
	ExtraData     []byte   `ssz-max:"32"`
	BaseFeePerGas *big.Int // Big.Int instead of the recommended uint256.Int
	BlockHash     Hash
	Transactions  [][]byte `ssz-max:"1048576,1073741824"`
}
