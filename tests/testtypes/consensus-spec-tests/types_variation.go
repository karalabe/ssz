// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import (
	"math/big"

	"github.com/prysmaticlabs/go-bitfield"
)

//go:generate go run -cover ../../../cmd/sszgen -type WithdrawalVariation -out gen_withdrawal_variation_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type HistoricalBatchVariation -out gen_historical_batch_variation_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type ExecutionPayloadVariation -out gen_execution_payload_variation_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type AttestationVariation1 -out gen_attestation_variation_1_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type AttestationVariation2 -out gen_attestation_variation_2_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type AttestationVariation3 -out gen_attestation_variation_3_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type AttestationDataVariation1 -out gen_attestation_data_variation_1_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type AttestationDataVariation2 -out gen_attestation_data_variation_2_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type AttestationDataVariation3 -out gen_attestation_data_variation_3_ssz.go

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

// The types below test that fork constraints generate correct code for runtime
// types (i.e. static objects embedded) for various positions.

type AttestationVariation1 struct {
	Future          uint64           `ssz-fork:"future"` // Currently unused field
	AggregationBits bitfield.Bitlist `ssz-max:"2048"`
	Data            *AttestationData
	Signature       [96]byte
}
type AttestationVariation2 struct {
	AggregationBits bitfield.Bitlist `ssz-max:"2048"`
	Data            *AttestationData
	Future          uint64 `ssz-fork:"future"` // Currently unused field
	Signature       [96]byte
}
type AttestationVariation3 struct {
	AggregationBits bitfield.Bitlist `ssz-max:"2048"`
	Data            *AttestationData
	Signature       [96]byte
	Future          uint64 `ssz-fork:"future"` // Currently unused field
}

type AttestationDataVariation1 struct {
	Future          uint64 `ssz-fork:"future"` // Currently unused field
	Slot            Slot
	Index           uint64
	BeaconBlockHash Hash
	Source          *Checkpoint
	Target          *Checkpoint
}
type AttestationDataVariation2 struct {
	Slot            Slot
	Index           uint64
	BeaconBlockHash Hash
	Future          uint64 `ssz-fork:"future"` // Currently unused field
	Source          *Checkpoint
	Target          *Checkpoint
}
type AttestationDataVariation3 struct {
	Slot            Slot
	Index           uint64
	BeaconBlockHash Hash
	Source          *Checkpoint
	Target          *Checkpoint
	Future          uint64 `ssz-fork:"future"` // Currently unused field
}
