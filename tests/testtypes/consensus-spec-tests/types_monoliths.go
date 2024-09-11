// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/holiman/uint256"

//go:generate go run -cover ../../../cmd/sszgen -type ExecutionPayloadMonolith -out gen_execution_payload_monolith_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type ExecutionPayloadHeaderMonolith -out gen_execution_payload_header_monolith_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type BeaconBlockBodyMonolith -out gen_beacon_block_body_monolith_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type BeaconStateMonolith -out gen_beacon_state_monolith_ssz.go

type BeaconBlockBodyMonolith struct {
	RandaoReveal          [96]byte
	Eth1Data              *Eth1Data
	Graffiti              [32]byte
	ProposerSlashings     []*ProposerSlashing           `ssz-max:"16"`
	AttesterSlashings     []*AttesterSlashing           `ssz-max:"2"`
	Attestations          []*Attestation                `ssz-max:"128"`
	Deposits              []*Deposit                    `ssz-max:"16"`
	VoluntaryExits        []*SignedVoluntaryExit        `ssz-max:"16"`
	SyncAggregate         *SyncAggregate                `               ssz-fork:"altair"`
	ExecutionPayload      *ExecutionPayloadMonolith     `               ssz-fork:"bellatrix"`
	BlsToExecutionChanges []*SignedBLSToExecutionChange `ssz-max:"16"   ssz-fork:"capella"`
	BlobKzgCommitments    [][48]byte                    `ssz-max:"4096" ssz-fork:"deneb"`
}

type BeaconStateMonolith struct {
	GenesisTime                  uint64
	GenesisValidatorsRoot        [32]byte
	Slot                         uint64
	Fork                         *Fork
	LatestBlockHeader            *BeaconBlockHeader
	BlockRoots                   [8192][32]byte
	StateRoots                   [8192][32]byte
	HistoricalRoots              [][32]byte `ssz-max:"16777216"`
	Eth1Data                     *Eth1Data
	Eth1DataVotes                []*Eth1Data `ssz-max:"2048"`
	Eth1DepositIndex             uint64
	Validators                   []*Validator `ssz-max:"1099511627776"`
	Balances                     []uint64     `ssz-max:"1099511627776"`
	RandaoMixes                  [65536][32]byte
	Slashings                    [8192]uint64
	PreviousEpochAttestations    []*PendingAttestation `ssz-max:"4096"          ssz-fork:"!altair"`
	CurrentEpochAttestations     []*PendingAttestation `ssz-max:"4096"          ssz-fork:"!altair"`
	PreviousEpochParticipation   []byte                `ssz-max:"1099511627776" ssz-fork:"altair"`
	CurrentEpochParticipation    []byte                `ssz-max:"1099511627776" ssz-fork:"altair"`
	JustificationBits            [1]byte               `ssz-size:"4" ssz:"bits"`
	PreviousJustifiedCheckpoint  *Checkpoint
	CurrentJustifiedCheckpoint   *Checkpoint
	FinalizedCheckpoint          *Checkpoint
	InactivityScores             []uint64                        `ssz-max:"1099511627776" ssz-fork:"altair"`
	CurrentSyncCommittee         *SyncCommittee                  `                        ssz-fork:"altair"`
	NextSyncCommittee            *SyncCommittee                  `                        ssz-fork:"altair"`
	LatestExecutionPayloadHeader *ExecutionPayloadHeaderMonolith `                        ssz-fork:"bellatrix"`
	NextWithdrawalIndex          *uint64                         `                        ssz-fork:"capella"`
	NextWithdrawalValidatorIndex *uint64                         `                        ssz-fork:"capella"`
	HistoricalSummaries          []*HistoricalSummary            `ssz-max:"16777216"      ssz-fork:"capella"`
}

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
	ExtraData     []byte `ssz-max:"32" ssz-fork:"frontier"`
	BaseFeePerGas *uint256.Int
	BlockHash     Hash
	Transactions  [][]byte      `ssz-max:"1048576,1073741824"`
	Withdrawals   []*Withdrawal `ssz-max:"16" ssz-fork:"shanghai"`
	BlobGasUsed   *uint64       `             ssz-fork:"cancun"`
	ExcessBlobGas *uint64       `             ssz-fork:"cancun"`
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
	ExtraData        []byte `ssz-max:"32" ssz-fork:"frontier"`
	BaseFeePerGas    [32]byte
	BlockHash        [32]byte
	TransactionsRoot [32]byte
	WithdrawalRoot   *[32]byte `ssz-fork:"shanghai"`
	BlobGasUsed      *uint64   `ssz-fork:"cancun"`
	ExcessBlobGas    *uint64   `ssz-fork:"cancun"`
}
