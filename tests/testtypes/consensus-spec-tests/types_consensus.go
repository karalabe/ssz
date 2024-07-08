// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/holiman/uint256"

//go:generate go run ../../../cmd/sszgen -type Checkpoint -out gen_checkpoint_ssz.go
//go:generate go run ../../../cmd/sszgen -type AttestationData -out gen_attestation_data_ssz.go
//go:generate go run ../../../cmd/sszgen -type BeaconBlockHeader -out gen_beacon_block_header_ssz.go
// go:generate go run ../../../cmd/sszgen -type Attestation -out gen_attestation_ssz.go
//go:generate go run ../../../cmd/sszgen -type DepositData -out gen_deposit_data_ssz.go
//go:generate go run ../../../cmd/sszgen -type Deposit -out gen_deposit_ssz.go
//go:generate go run ../../../cmd/sszgen -type Eth1Data -out gen_eth1_data_ssz.go
// go:generate go run ../../../cmd/sszgen -type ExecutionPayload -out gen_execution_payload_ssz.go
//go:generate go run ../../../cmd/sszgen -type HistoricalBatch -out gen_historical_batch_ssz.go
//go:generate go run ../../../cmd/sszgen -type ProposerSlashing -out gen_proposed_slashing_ssz.go
//go:generate go run ../../../cmd/sszgen -type SignedBeaconBlockHeader -out gen_signed_beacon_block_header_ssz.go
//go:generate go run ../../../cmd/sszgen -type SignedVoluntaryExit -out gen_signed_voluntary_exit_ssz.go
//go:generate go run ../../../cmd/sszgen -type VoluntaryExit -out gen_voluntary_exit_ssz.go
//go:generate go run ../../../cmd/sszgen -type Validator -out gen_validator_ssz.go
//go:generate go run ../../../cmd/sszgen -type Withdrawal -out gen_withdrawal_ssz.go

type Attestation struct {
	AggregationBits []byte `ssz-max:"2048"`
	Data            *AttestationData
	Signature       [96]byte
}

type AttestationData struct {
	Slot            Slot
	Index           uint64
	BeaconBlockHash Hash
	Source          *Checkpoint
	Target          *Checkpoint
}

type BeaconBlockHeader struct {
	Slot          uint64
	ProposerIndex uint64
	ParentRoot    Hash
	StateRoot     Hash
	BodyRoot      Hash
}

type Checkpoint struct {
	Epoch uint64
	Root  Hash
}

type Deposit struct {
	Proof [33][32]byte
	Data  *DepositData
}

type DepositData struct {
	Pubkey                [48]byte
	WithdrawalCredentials [32]byte
	Amount                uint64
	Signature             [96]byte
}

type Eth1Data struct {
	DepositRoot  Hash
	DepositCount uint64
	BlockHash    Hash
}

type ExecutionPayload struct {
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
	Transactions  [][]byte `ssz-max:"1048576"`
}

type HistoricalBatch struct {
	BlockRoots [8192]Hash
	StateRoots [8192]Hash
}

type ProposerSlashing struct {
	Header1 *SignedBeaconBlockHeader
	Header2 *SignedBeaconBlockHeader
}

type SignedBeaconBlockHeader struct {
	Header    *BeaconBlockHeader
	Signature [96]byte
}

type SignedVoluntaryExit struct {
	Exit      *VoluntaryExit
	Signature [96]byte
}

type VoluntaryExit struct {
	Epoch          uint64
	ValidatorIndex uint64
}

type Validator struct {
	Pubkey                     [48]byte
	WithdrawalCredentials      [32]byte
	EffectiveBalance           uint64
	Slashed                    bool
	ActivationEligibilityEpoch uint64
	ActivationEpoch            uint64
	ExitEpoch                  uint64
	WithdrawableEpoch          uint64
}

type Withdrawal struct {
	Index     uint64
	Validator uint64
	Address   Address
	Amount    uint64
}
