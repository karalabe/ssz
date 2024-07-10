// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import (
	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
)

//go:generate go run ../../../cmd/sszgen -type Checkpoint -out gen_checkpoint_ssz.go
//go:generate go run ../../../cmd/sszgen -type AttestationData -out gen_attestation_data_ssz.go
//go:generate go run ../../../cmd/sszgen -type BeaconBlockHeader -out gen_beacon_block_header_ssz.go
//go:generate go run ../../../cmd/sszgen -type BLSToExecutionChange -out gen_bls_to_execution_change_ssz.go
//go:generate go run ../../../cmd/sszgen -type Attestation -out gen_attestation_ssz.go
//go:generate go run ../../../cmd/sszgen -type AggregateAndProof -out gen_aggregate_and_proof_ssz.go
//go:generate go run ../../../cmd/sszgen -type DepositData -out gen_deposit_data_ssz.go
//go:generate go run ../../../cmd/sszgen -type DepositMessage -out gen_deposit_message_ssz.go
//go:generate go run ../../../cmd/sszgen -type Deposit -out gen_deposit_ssz.go
//go:generate go run ../../../cmd/sszgen -type Eth1Block -out gen_eth1_block_ssz.go
//go:generate go run ../../../cmd/sszgen -type Eth1Data -out gen_eth1_data_ssz.go
//go:generate go run ../../../cmd/sszgen -type ExecutionPayload -out gen_execution_payload_ssz.go
//go:generate go run ../../../cmd/sszgen -type ExecutionPayloadHeader -out gen_execution_payload_header_ssz.go
//go:generate go run ../../../cmd/sszgen -type Fork -out gen_fork_ssz.go
//go:generate go run ../../../cmd/sszgen -type HistoricalBatch -out gen_historical_batch_ssz.go
//go:generate go run ../../../cmd/sszgen -type HistoricalSummary -out gen_historical_summary_ssz.go
//go:generate go run ../../../cmd/sszgen -type IndexedAttestation -out gen_indexed_attestation_ssz.go
//go:generate go run ../../../cmd/sszgen -type AttesterSlashing -out gen_attester_slashing_ssz.go
//go:generate go run ../../../cmd/sszgen -type PendingAttestation -out gen_pending_attestation_ssz.go
//go:generate go run ../../../cmd/sszgen -type SignedBeaconBlockHeader -out gen_signed_beacon_block_header_ssz.go
//go:generate go run ../../../cmd/sszgen -type ProposerSlashing -out gen_proposer_slashing_ssz.go
//go:generate go run ../../../cmd/sszgen -type SignedBLSToExecutionChange -out gen_signed_bls_to_execution_change_ssz.go
//go:generate go run ../../../cmd/sszgen -type SyncAggregate -out gen_sync_aggregate_ssz.go
//go:generate go run ../../../cmd/sszgen -type SyncCommittee -out gen_sync_committee_ssz.go
//go:generate go run ../../../cmd/sszgen -type VoluntaryExit -out gen_voluntary_exit_ssz.go
//go:generate go run ../../../cmd/sszgen -type SignedVoluntaryExit -out gen_signed_voluntary_exit_ssz.go
//go:generate go run ../../../cmd/sszgen -type Validator -out gen_validator_ssz.go
//go:generate go run ../../../cmd/sszgen -type Withdrawal -out gen_withdrawal_ssz.go
//go:generate go run ../../../cmd/sszgen -type ExecutionPayloadCapella -out gen_execution_payload_capella_ssz.go
//go:generate go run ../../../cmd/sszgen -type ExecutionPayloadHeaderCapella -out gen_execution_payload_header_capella_ssz.go
//go:generate go run ../../../cmd/sszgen -type ExecutionPayloadDeneb -out gen_execution_payload_deneb_ssz.go
//go:generate go run ../../../cmd/sszgen -type ExecutionPayloadHeaderDeneb -out gen_execution_payload_header_deneb_ssz.go
//go:generate go run ../../../cmd/sszgen -type BeaconState -out gen_beacon_state_ssz.go
//go:generate go run ../../../cmd/sszgen -type BeaconBlockBody -out gen_beacon_block_body_ssz.go
//go:generate go run ../../../cmd/sszgen -type BeaconBlockBodyAltair -out gen_beacon_block_body_altair_ssz.go
//go:generate go run ../../../cmd/sszgen -type BeaconBlockBodyBellatrix -out gen_beacon_block_body_bellatrix_ssz.go
//go:generate go run ../../../cmd/sszgen -type BeaconBlockBodyCapella -out gen_beacon_block_body_capella_ssz.go
//go:generate go run ../../../cmd/sszgen -type BeaconBlockBodyDeneb -out gen_beacon_block_body_deneb_ssz.go
//go:generate go run ../../../cmd/sszgen -type BeaconBlock -out gen_beacon_block_ssz.go

// Slot is an alias of uint64
type Slot uint64

// Hash is a standalone mock of go-ethereum;s common.Hash
type Hash [32]byte

// Address is a standalone mock of go-ethereum's common.Address
type Address [20]byte

// LogsBloom is a standalone mock of go-ethereum's types.LogsBloom
type LogsBloom [256]byte

// Roots is a helper type to foce a generator quirk.
type Roots [8192]Hash

type AggregateAndProof struct {
	Index          uint64
	Aggregate      *Attestation
	SelectionProof [96]byte
}

type Attestation struct {
	AggregationBits bitfield.Bitlist `ssz-max:"2048"`
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

type AttesterSlashing struct {
	Attestation1 *IndexedAttestation
	Attestation2 *IndexedAttestation
}

type BeaconBlock struct {
	Slot          Slot
	ProposerIndex uint64
	ParentRoot    Hash
	StateRoot     Hash
	Body          *BeaconBlockBody
}

type BeaconBlockHeader struct {
	Slot          uint64
	ProposerIndex uint64
	ParentRoot    Hash
	StateRoot     Hash
	BodyRoot      Hash
}

type BeaconBlockBody struct {
	RandaoReveal      [96]byte
	Eth1Data          *Eth1Data
	Graffiti          [32]byte
	ProposerSlashings []*ProposerSlashing    `ssz-max:"16"`
	AttesterSlashings []*AttesterSlashing    `ssz-max:"2"`
	Attestations      []*Attestation         `ssz-max:"128"`
	Deposits          []*Deposit             `ssz-max:"16"`
	VoluntaryExits    []*SignedVoluntaryExit `ssz-max:"16"`
}

type BeaconBlockBodyAltair struct {
	RandaoReveal      [96]byte
	Eth1Data          *Eth1Data
	Graffiti          [32]byte
	ProposerSlashings []*ProposerSlashing    `ssz-max:"16"`
	AttesterSlashings []*AttesterSlashing    `ssz-max:"2"`
	Attestations      []*Attestation         `ssz-max:"128"`
	Deposits          []*Deposit             `ssz-max:"16"`
	VoluntaryExits    []*SignedVoluntaryExit `ssz-max:"16"`
	SyncAggregate     *SyncAggregate
}

type BeaconBlockBodyBellatrix struct {
	RandaoReveal      [96]byte
	Eth1Data          *Eth1Data
	Graffiti          [32]byte
	ProposerSlashings []*ProposerSlashing    `ssz-max:"16"`
	AttesterSlashings []*AttesterSlashing    `ssz-max:"2"`
	Attestations      []*Attestation         `ssz-max:"128"`
	Deposits          []*Deposit             `ssz-max:"16"`
	VoluntaryExits    []*SignedVoluntaryExit `ssz-max:"16"`
	SyncAggregate     *SyncAggregate
	ExecutionPayload  *ExecutionPayload
}

type BeaconBlockBodyCapella struct {
	RandaoReveal          [96]byte
	Eth1Data              *Eth1Data
	Graffiti              [32]byte
	ProposerSlashings     []*ProposerSlashing    `ssz-max:"16"`
	AttesterSlashings     []*AttesterSlashing    `ssz-max:"2"`
	Attestations          []*Attestation         `ssz-max:"128"`
	Deposits              []*Deposit             `ssz-max:"16"`
	VoluntaryExits        []*SignedVoluntaryExit `ssz-max:"16"`
	SyncAggregate         *SyncAggregate
	ExecutionPayload      *ExecutionPayloadCapella
	BlsToExecutionChanges []*SignedBLSToExecutionChange `ssz-max:"16"`
}

type BeaconBlockBodyDeneb struct {
	RandaoReveal          [96]byte
	Eth1Data              *Eth1Data
	Graffiti              [32]byte
	ProposerSlashings     []*ProposerSlashing    `ssz-max:"16"`
	AttesterSlashings     []*AttesterSlashing    `ssz-max:"2"`
	Attestations          []*Attestation         `ssz-max:"128"`
	Deposits              []*Deposit             `ssz-max:"16"`
	VoluntaryExits        []*SignedVoluntaryExit `ssz-max:"16"`
	SyncAggregate         *SyncAggregate
	ExecutionPayload      *ExecutionPayloadDeneb
	BlsToExecutionChanges []*SignedBLSToExecutionChange `ssz-max:"16"`
	BlobKzgCommitments    [][48]byte                    `ssz-max:"4096"`
}

type BeaconState struct {
	GenesisTime                 uint64
	GenesisValidatorsRoot       [32]byte
	Slot                        uint64
	Fork                        *Fork
	LatestBlockHeader           *BeaconBlockHeader
	BlockRoots                  [8192][32]byte
	StateRoots                  [8192][32]byte
	HistoricalRoots             [][32]byte `ssz-max:"16777216"`
	Eth1Data                    *Eth1Data
	Eth1DataVotes               []*Eth1Data `ssz-max:"2048"`
	Eth1DepositIndex            uint64
	Validators                  []*Validator `ssz-max:"1099511627776"`
	Balances                    []uint64     `ssz-max:"1099511627776"`
	RandaoMixes                 [65536][32]byte
	Slashings                   [8192]uint64
	PreviousEpochAttestations   []*PendingAttestation `ssz-max:"4096"`
	CurrentEpochAttestations    []*PendingAttestation `ssz-max:"4096"`
	JustificationBits           [1]byte               `ssz-size:"4" ssz:"bits"`
	PreviousJustifiedCheckpoint *Checkpoint
	CurrentJustifiedCheckpoint  *Checkpoint
	FinalizedCheckpoint         *Checkpoint
}

type BLSToExecutionChange struct {
	ValidatorIndex     uint64
	FromBLSPubKey      [48]byte
	ToExecutionAddress [20]byte
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

type DepositMessage struct {
	Pubkey                [48]byte
	WithdrawalCredentials [32]byte
	Amount                uint64
}

type Eth1Block struct {
	Timestamp    uint64
	DepositRoot  [32]byte
	DepositCount uint64
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
	Transactions  [][]byte `ssz-max:"1048576,1073741824"`
}

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
	ExtraData     []byte `ssz-max:"32"`
	BaseFeePerGas *uint256.Int
	BlockHash     Hash
	Transactions  [][]byte      `ssz-max:"1048576,1073741824"`
	Withdrawals   []*Withdrawal `ssz-max:"16"`
}

type ExecutionPayloadDeneb struct {
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
	Withdrawals   []*Withdrawal `ssz-max:"16"`
	BlobGasUsed   uint64
	ExcessBlobGas uint64
}

type ExecutionPayloadHeader struct {
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
}

type ExecutionPayloadHeaderCapella struct {
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
	WithdrawalRoot   [32]byte
}

type ExecutionPayloadHeaderDeneb struct {
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
	WithdrawalRoot   [32]byte
	BlobGasUsed      uint64
	ExcessBlobGas    uint64
}

type Fork struct {
	PreviousVersion [4]byte
	CurrentVersion  [4]byte
	Epoch           uint64
}

type HistoricalBatch struct {
	BlockRoots [8192]Hash
	StateRoots Roots
}

type HistoricalSummary struct {
	BlockSummaryRoot [32]byte
	StateSummaryRoot [32]byte
}

type IndexedAttestation struct {
	AttestationIndices []uint64 `ssz-max:"2048"`
	Data               *AttestationData
	Signature          [96]byte
}

type PendingAttestation struct {
	AggregationBits bitfield.Bitlist `ssz-max:"2048"`
	Data            *AttestationData
	InclusionDelay  uint64
	ProposerIndex   uint64
}

type ProposerSlashing struct {
	Header1 *SignedBeaconBlockHeader
	Header2 *SignedBeaconBlockHeader
}

type SignedBeaconBlockHeader struct {
	Header    *BeaconBlockHeader
	Signature [96]byte
}

type SignedBLSToExecutionChange struct {
	Message   *BLSToExecutionChange
	Signature [96]byte
}

type SignedVoluntaryExit struct {
	Exit      *VoluntaryExit
	Signature [96]byte
}

type SyncAggregate struct {
	SyncCommiteeBits      [64]byte
	SyncCommiteeSignature [96]byte
}

type SyncCommittee struct {
	PubKeys         [512][48]byte
	AggregatePubKey [48]byte
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
