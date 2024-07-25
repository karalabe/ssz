// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/holiman/uint256"

//go:generate go run ../../../cmd/sszgen -type ExecutionPayloadMonolith -out gen_execution_payload_monolith_ssz.go
//go:generate go run ../../../cmd/sszgen -type ExecutionPayloadHeaderMonolith -out gen_execution_payload_header_monolith_ssz.go

type ExecutionPayloadMonolith struct {
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
	ExtraData     []byte `ssz-max:"32"`
	BaseFeePerGas *uint256.Int
	BlockHash     Hash
	Transactions  [][]byte      `ssz-max:"1048576,1073741824"`
	Withdrawals   []*Withdrawal `ssz-max:"16" ssz-fork:"capella"`
	BlobGasUsed   uint64        `ssz-fork:"deneb"`
	ExcessBlobGas uint64        `ssz-fork:"deneb"`
}

type ExecutionPayloadHeaderMonolith struct {
	ParentHash       [32]byte
	FeeRecipient     [20]byte
	StateRoot        [32]byte
	ReceiptsRoot     [32]byte
	LogsBloom        [256]byte
	PrevRandao       [32]byte
	BlockNumber      uint64
	GasLimit         uint64
	GasUsed          uint64
	Timestamp        uint64
	ExtraData        []byte `ssz-max:"32"`
	BaseFeePerGas    [32]byte
	BlockHash        [32]byte
	TransactionsRoot [32]byte
	WithdrawalRoot   [32]byte `ssz-fork:"capella"`
	BlobGasUsed      uint64   `ssz-fork:"deneb"`
	ExcessBlobGas    uint64   `ssz-fork:"deneb"`
}
