// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package tests

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/golang/snappy"
	"github.com/karalabe/ssz"
	types "github.com/karalabe/ssz/tests/testtypes/consensus-spec-tests"
	"gopkg.in/yaml.v3"
)

var (
	// consensusSpecTestsRoot is the folder where the consensus ssz tests are located.
	consensusSpecTestsRoot = filepath.Join("testdata", "consensus-spec-tests", "tests", "mainnet")

	// consensusSpecTestsDone tracks which types have had their tests ran, so all the
	// untested stuff can fail noisily.
	consensusSpecTestsDone = make(map[string]map[string]struct{})
	consensusSpecTestsLock sync.Mutex
)

// commonPrefix returns the common prefix in two byte slices.
func commonPrefix(a []byte, b []byte) []byte {
	var prefix []byte

	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		prefix = append(prefix, a[0])
		a, b = a[1:], b[1:]
	}
	return prefix
}

// TestConsensusSpecs iterates over all the (supported) consensus SSZ types and
// runs the encoding/decoding/hashing round.
func TestConsensusSpecs(t *testing.T) {
	testConsensusSpecType[*types.AggregateAndProof](t, "AggregateAndProof", "altair", "bellatrix", "capella", "deneb", "eip7594", "phase0", "whisk")
	testConsensusSpecType[*types.Attestation](t, "Attestation", "altair", "bellatrix", "capella", "deneb", "eip7594", "phase0", "whisk")
	testConsensusSpecType[*types.AttestationData](t, "AttestationData")
	testConsensusSpecType[*types.AttesterSlashing](t, "AttesterSlashing", "phase0", "altair", "bellatrix", "capella", "deneb")
	testConsensusSpecType[*types.BeaconBlock](t, "BeaconBlock", "phase0")
	testConsensusSpecType[*types.BeaconBlockBody](t, "BeaconBlockBody", "phase0")
	testConsensusSpecType[*types.BeaconBlockBodyAltair](t, "BeaconBlockBody", "altair")
	testConsensusSpecType[*types.BeaconBlockBodyBellatrix](t, "BeaconBlockBody", "bellatrix")
	testConsensusSpecType[*types.BeaconBlockBodyCapella](t, "BeaconBlockBody", "capella")
	testConsensusSpecType[*types.BeaconBlockBodyDeneb](t, "BeaconBlockBody", "deneb", "eip7594")
	testConsensusSpecType[*types.BeaconBlockHeader](t, "BeaconBlockHeader")
	testConsensusSpecType[*types.BeaconState](t, "BeaconState", "phase0")
	testConsensusSpecType[*types.BeaconStateCapella](t, "BeaconState", "capella")
	testConsensusSpecType[*types.BeaconStateDeneb](t, "BeaconState", "deneb")
	testConsensusSpecType[*types.BLSToExecutionChange](t, "BLSToExecutionChange")
	testConsensusSpecType[*types.Checkpoint](t, "Checkpoint")
	testConsensusSpecType[*types.Deposit](t, "Deposit")
	testConsensusSpecType[*types.DepositData](t, "DepositData")
	testConsensusSpecType[*types.DepositMessage](t, "DepositMessage")
	testConsensusSpecType[*types.Eth1Block](t, "Eth1Block")
	testConsensusSpecType[*types.Eth1Data](t, "Eth1Data")
	testConsensusSpecType[*types.ExecutionPayload](t, "ExecutionPayload", "bellatrix")
	testConsensusSpecType[*types.ExecutionPayloadHeader](t, "ExecutionPayloadHeader", "bellatrix")
	testConsensusSpecType[*types.ExecutionPayloadCapella](t, "ExecutionPayload", "capella")
	testConsensusSpecType[*types.ExecutionPayloadHeaderCapella](t, "ExecutionPayloadHeader", "capella")
	testConsensusSpecType[*types.ExecutionPayloadDeneb](t, "ExecutionPayload", "deneb", "eip7594")
	testConsensusSpecType[*types.ExecutionPayloadHeaderDeneb](t, "ExecutionPayloadHeader", "deneb", "eip7594")
	testConsensusSpecType[*types.Fork](t, "Fork")
	testConsensusSpecType[*types.HistoricalBatch](t, "HistoricalBatch")
	testConsensusSpecType[*types.HistoricalSummary](t, "HistoricalSummary")
	testConsensusSpecType[*types.IndexedAttestation](t, "IndexedAttestation", "phase0", "altair", "bellatrix", "capella", "deneb")
	testConsensusSpecType[*types.PendingAttestation](t, "PendingAttestation")
	testConsensusSpecType[*types.ProposerSlashing](t, "ProposerSlashing")
	testConsensusSpecType[*types.SignedBeaconBlockHeader](t, "SignedBeaconBlockHeader")
	testConsensusSpecType[*types.SignedBLSToExecutionChange](t, "SignedBLSToExecutionChange")
	testConsensusSpecType[*types.SignedVoluntaryExit](t, "SignedVoluntaryExit")
	testConsensusSpecType[*types.SyncAggregate](t, "SyncAggregate")
	testConsensusSpecType[*types.SyncCommittee](t, "SyncCommittee")
	testConsensusSpecType[*types.Validator](t, "Validator")
	testConsensusSpecType[*types.VoluntaryExit](t, "VoluntaryExit")
	testConsensusSpecType[*types.Withdrawal](t, "Withdrawal")

	// Add some API variations to test different codec implementations
	testConsensusSpecType[*types.ExecutionPayloadVariation](t, "ExecutionPayload", "bellatrix")
	testConsensusSpecType[*types.HistoricalBatchVariation](t, "HistoricalBatch")
	testConsensusSpecType[*types.WithdrawalVariation](t, "Withdrawal")

	// Iterate over all the untouched tests and report them
	// 	forks, err := os.ReadDir(consensusSpecTestsRoot)
	//	if err != nil {
	//		t.Fatalf("failed to walk fork collection: %v", err)
	//	}
	//	for _, fork := range forks {
	//		if _, ok := consensusSpecTestsDone[fork.Name()]; !ok {
	//			t.Errorf("no tests ran for %v", fork.Name())
	//			continue
	//		}
	//		types, err := os.ReadDir(filepath.Join(consensusSpecTestsRoot, fork.Name(), "ssz_static"))
	//		if err != nil {
	//			t.Fatalf("failed to walk type collection of %v: %v", fork, err)
	//		}
	//		for _, kind := range types {
	//			if _, ok := consensusSpecTestsDone[fork.Name()][kind.Name()]; !ok {
	//				t.Errorf("no tests ran for %v/%v", fork.Name(), kind.Name())
	//			}
	//		}
	//	}
}

// newableObject is a generic type whose purpose is to enforce that ssz.Object
// is specifically implemented on a struct pointer. That's needed to allow us
// to instantiate new structs via `new` when parsing.
type newableObject[U any] interface {
	ssz.Object
	*U
}

func testConsensusSpecType[T newableObject[U], U any](t *testing.T, kind string, forks ...string) {
	// If no fork was specified, iterate over all of them and use the same type
	if len(forks) == 0 {
		forks, err := os.ReadDir(consensusSpecTestsRoot)
		if err != nil {
			t.Errorf("failed to walk spec collection %v: %v", consensusSpecTestsRoot, err)
			return
		}
		for _, fork := range forks {
			if _, err := os.Stat(filepath.Join(consensusSpecTestsRoot, fork.Name(), "ssz_static", kind, "ssz_random")); err == nil {
				testConsensusSpecType[T, U](t, kind, fork.Name())
			}
		}
		return
	}
	// Some specific fork was requested, look that up explicitly
	for _, fork := range forks {
		path := filepath.Join(consensusSpecTestsRoot, fork, "ssz_static", kind, "ssz_random")

		tests, err := os.ReadDir(path)
		if err != nil {
			t.Errorf("failed to walk test collection %v: %v", path, err)
			return
		}
		// Track this test suite done, whether succeeds of fails is irrelevant
		consensusSpecTestsLock.Lock()
		if _, ok := consensusSpecTestsDone[fork]; !ok {
			consensusSpecTestsDone[fork] = make(map[string]struct{})
		}
		consensusSpecTestsDone[fork][kind] = struct{}{}
		consensusSpecTestsLock.Unlock()

		// Run all the subtests found in the folder
		for _, test := range tests {
			t.Run(fmt.Sprintf("%s/%s/%s", fork, kind, test.Name()), func(t *testing.T) {
				// Parse the input SSZ data and the expected root for the test
				inSnappy, err := os.ReadFile(filepath.Join(path, test.Name(), "serialized.ssz_snappy"))
				if err != nil {
					t.Fatalf("failed to load snapy ssz binary: %v", err)
				}
				inSSZ, err := snappy.Decode(nil, inSnappy)
				if err != nil {
					t.Fatalf("failed to parse snappy ssz binary: %v", err)
				}
				inYAML, err := os.ReadFile(filepath.Join(path, test.Name(), "roots.yaml"))
				if err != nil {
					t.Fatalf("failed to load yaml root: %v", err)
				}
				inRoot := struct {
					Root string `yaml:"root"`
				}{}
				if err = yaml.Unmarshal(inYAML, &inRoot); err != nil {
					t.Fatalf("failed to parse yaml root: %v", err)
				}
				// Do a decode/encode round. Would be nicer to parse out the value
				// from yaml and check that too, but hex-in-yaml makes everything
				// beyond annoying. C'est la vie.
				obj := T(new(U))
				if err := ssz.DecodeFromStream(bytes.NewReader(inSSZ), obj, uint32(len(inSSZ))); err != nil {
					t.Fatalf("failed to decode SSZ stream: %v", err)
				}
				blob := new(bytes.Buffer)
				if err := ssz.EncodeToStream(blob, obj); err != nil {
					t.Fatalf("failed to re-encode SSZ stream: %v", err)
				}
				if !bytes.Equal(blob.Bytes(), inSSZ) {
					prefix := commonPrefix(blob.Bytes(), inSSZ)
					t.Fatalf("re-encoded stream mismatch: have %x, want %x, common prefix %d, have left %x, want left %x",
						blob, inSSZ, len(prefix), blob.Bytes()[len(prefix):], inSSZ[len(prefix):])
				}
				obj = T(new(U))
				if err := ssz.DecodeFromBytes(inSSZ, obj); err != nil {
					t.Fatalf("failed to decode SSZ buffer: %v", err)
				}
				bin := make([]byte, ssz.Size(obj))
				if err := ssz.EncodeToBytes(bin, obj); err != nil {
					t.Fatalf("failed to re-encode SSZ buffer: %v", err)
				}
				if !bytes.Equal(bin, inSSZ) {
					prefix := commonPrefix(bin, inSSZ)
					t.Fatalf("re-encoded bytes mismatch: have %x, want %x, common prefix %d, have left %x, want left %x",
						blob, inSSZ, len(prefix), bin[len(prefix):], inSSZ[len(prefix):])
				}
				// Encoder/decoder seems to work, check if the size reported by the
				// encoded object actually matches the encoded stream
				if size := ssz.Size(obj); size != uint32(len(inSSZ)) {
					t.Fatalf("reported/generated size mismatch: reported %v, generated %v", size, len(inSSZ))
				}
				hash := ssz.HashSequential(obj)
				if fmt.Sprintf("%#x", hash) != inRoot.Root {
					t.Fatalf("sequential merkle root mismatch: have %#x, want %s", hash, inRoot.Root)
				}
				hash = ssz.HashConcurrent(obj)
				if fmt.Sprintf("%#x", hash) != inRoot.Root {
					t.Fatalf("concurrent merkle root mismatch: have %#x, want %s", hash, inRoot.Root)
				}
			})
		}
	}
}

// BenchmarkConsensusSpecs iterates over all the (supported) consensus SSZ types and
// runs the encoding/decoding/hashing benchmark round.
func BenchmarkConsensusSpecs(b *testing.B) {
	benchmarkConsensusSpecType[*types.ExecutionPayloadVariation](b, "bellatrix", "ExecutionPayload")

	benchmarkConsensusSpecType[*types.AggregateAndProof](b, "deneb", "AggregateAndProof")
	benchmarkConsensusSpecType[*types.Attestation](b, "deneb", "Attestation")
	benchmarkConsensusSpecType[*types.AttestationData](b, "deneb", "AttestationData")
	benchmarkConsensusSpecType[*types.AttesterSlashing](b, "deneb", "AttesterSlashing")
	benchmarkConsensusSpecType[*types.BeaconBlock](b, "phase0", "BeaconBlock")
	benchmarkConsensusSpecType[*types.BeaconBlockBodyDeneb](b, "deneb", "BeaconBlockBody")
	benchmarkConsensusSpecType[*types.BeaconBlockHeader](b, "deneb", "BeaconBlockHeader")
	benchmarkConsensusSpecType[*types.BeaconState](b, "phase0", "BeaconState")
	benchmarkConsensusSpecType[*types.BLSToExecutionChange](b, "deneb", "BLSToExecutionChange")
	benchmarkConsensusSpecType[*types.Checkpoint](b, "deneb", "Checkpoint")
	benchmarkConsensusSpecType[*types.Deposit](b, "deneb", "Deposit")
	benchmarkConsensusSpecType[*types.DepositData](b, "deneb", "DepositData")
	benchmarkConsensusSpecType[*types.DepositMessage](b, "deneb", "DepositMessage")
	benchmarkConsensusSpecType[*types.Eth1Block](b, "deneb", "Eth1Block")
	benchmarkConsensusSpecType[*types.Eth1Data](b, "deneb", "Eth1Data")
	benchmarkConsensusSpecType[*types.ExecutionPayloadDeneb](b, "deneb", "ExecutionPayload")
	benchmarkConsensusSpecType[*types.ExecutionPayloadHeaderDeneb](b, "deneb", "ExecutionPayloadHeader")
	benchmarkConsensusSpecType[*types.Fork](b, "deneb", "Fork")
	benchmarkConsensusSpecType[*types.HistoricalBatch](b, "deneb", "HistoricalBatch")
	benchmarkConsensusSpecType[*types.HistoricalSummary](b, "deneb", "HistoricalSummary")
	benchmarkConsensusSpecType[*types.IndexedAttestation](b, "deneb", "IndexedAttestation")
	benchmarkConsensusSpecType[*types.PendingAttestation](b, "deneb", "PendingAttestation")
	benchmarkConsensusSpecType[*types.ProposerSlashing](b, "deneb", "ProposerSlashing")
	benchmarkConsensusSpecType[*types.SignedBeaconBlockHeader](b, "deneb", "SignedBeaconBlockHeader")
	benchmarkConsensusSpecType[*types.SignedBLSToExecutionChange](b, "deneb", "SignedBLSToExecutionChange")
	benchmarkConsensusSpecType[*types.SignedVoluntaryExit](b, "deneb", "SignedVoluntaryExit")
	benchmarkConsensusSpecType[*types.SyncAggregate](b, "deneb", "SyncAggregate")
	benchmarkConsensusSpecType[*types.SyncCommittee](b, "deneb", "SyncCommittee")
	benchmarkConsensusSpecType[*types.Validator](b, "deneb", "Validator")
	benchmarkConsensusSpecType[*types.VoluntaryExit](b, "deneb", "VoluntaryExit")
	benchmarkConsensusSpecType[*types.Withdrawal](b, "deneb", "Withdrawal")
}

func benchmarkConsensusSpecType[T newableObject[U], U any](b *testing.B, fork, kind string) {
	path := filepath.Join(consensusSpecTestsRoot, fork, "ssz_static", kind, "ssz_random", "case_4")

	// Parse the input SSZ data for this specific dataset and decode it
	inSnappy, err := os.ReadFile(filepath.Join(path, "serialized.ssz_snappy"))
	if err != nil {
		b.Fatalf("failed to load snapy ssz binary: %v", err)
	}
	inSSZ, err := snappy.Decode(nil, inSnappy)
	if err != nil {
		b.Fatalf("failed to parse snappy ssz binary: %v", err)
	}
	inObj := T(new(U))
	if err := ssz.DecodeFromStream(bytes.NewReader(inSSZ), inObj, uint32(len(inSSZ))); err != nil {
		b.Fatalf("failed to decode SSZ stream: %v", err)
	}
	// Start the benchmarks for all the different operations
	b.Run(fmt.Sprintf("%s/encode-stream", kind), func(b *testing.B) {
		b.SetBytes(int64(len(inSSZ)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err := ssz.EncodeToStream(io.Discard, inObj); err != nil {
				b.Fatalf("failed to encode SSZ stream: %v", err)
			}
		}
	})
	b.Run(fmt.Sprintf("%s/encode-buffer", kind), func(b *testing.B) {
		blob := make([]byte, len(inSSZ))

		b.SetBytes(int64(len(inSSZ)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err := ssz.EncodeToBytes(blob, inObj); err != nil {
				b.Fatalf("failed to encode SSZ bytes: %v", err)
			}
		}
	})
	b.Run(fmt.Sprintf("%s/decode-stream", kind), func(b *testing.B) {
		obj := T(new(U))
		r := bytes.NewReader(inSSZ)

		b.SetBytes(int64(len(inSSZ)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err := ssz.DecodeFromStream(r, obj, uint32(len(inSSZ))); err != nil {
				b.Fatalf("failed to decode SSZ stream: %v", err)
			}
			r.Reset(inSSZ)
		}
	})
	b.Run(fmt.Sprintf("%s/decode-buffer", kind), func(b *testing.B) {
		obj := T(new(U))

		b.SetBytes(int64(len(inSSZ)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err := ssz.DecodeFromBytes(inSSZ, obj); err != nil {
				b.Fatalf("failed to decode SSZ stream: %v", err)
			}
		}
	})
	b.Run(fmt.Sprintf("%s/merkleize-sequential", kind), func(b *testing.B) {
		obj := T(new(U))
		if err := ssz.DecodeFromBytes(inSSZ, obj); err != nil {
			b.Fatalf("failed to decode SSZ stream: %v", err)
		}
		b.SetBytes(int64(len(inSSZ)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ssz.HashSequential(obj)
		}
	})
	b.Run(fmt.Sprintf("%s/merkleize-concurrent", kind), func(b *testing.B) {
		obj := T(new(U))
		if err := ssz.DecodeFromBytes(inSSZ, obj); err != nil {
			b.Fatalf("failed to decode SSZ stream: %v", err)
		}
		b.SetBytes(int64(len(inSSZ)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ssz.HashConcurrent(obj)
		}
	})
}

// Various fuzz targets can be found below, one for each consensus spec type. The
// methods will start by feeding all the consensus spec test data and then will do
// infinite decoding runs. Anything that succeeds will get re-encoded, re-decoded,
// etc. to test different functions.

func FuzzConsensusSpecsAggregateAndProof(f *testing.F) {
	fuzzConsensusSpecType[*types.AggregateAndProof](f, "AggregateAndProof")
}
func FuzzConsensusSpecsAttestation(f *testing.F) {
	fuzzConsensusSpecType[*types.Attestation](f, "Attestation")
}
func FuzzConsensusSpecsAttestationData(f *testing.F) {
	fuzzConsensusSpecType[*types.AttestationData](f, "AttestationData")
}
func FuzzConsensusSpecsAttesterSlashing(f *testing.F) {
	fuzzConsensusSpecType[*types.AttesterSlashing](f, "AttesterSlashing")
}
func FuzzConsensusSpecsBeaconBlock(f *testing.F) {
	fuzzConsensusSpecType[*types.BeaconBlock](f, "BeaconBlock")
}
func FuzzConsensusSpecsBeaconBlockBody(f *testing.F) {
	fuzzConsensusSpecType[*types.BeaconBlockBody](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockBodyAltair(f *testing.F) {
	fuzzConsensusSpecType[*types.BeaconBlockBodyAltair](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockBodyBellatrix(f *testing.F) {
	fuzzConsensusSpecType[*types.BeaconBlockBodyBellatrix](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockBodyCapella(f *testing.F) {
	fuzzConsensusSpecType[*types.BeaconBlockBodyCapella](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockBodyDeneb(f *testing.F) {
	fuzzConsensusSpecType[*types.BeaconBlockBodyDeneb](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockHeader(f *testing.F) {
	fuzzConsensusSpecType[*types.BeaconBlockHeader](f, "BeaconBlockHeader")
}
func FuzzConsensusSpecsBeaconState(f *testing.F) {
	fuzzConsensusSpecType[*types.BeaconState](f, "BeaconState")
}
func FuzzConsensusSpecsBLSToExecutionChange(f *testing.F) {
	fuzzConsensusSpecType[*types.BLSToExecutionChange](f, "BLSToExecutionChange")
}
func FuzzConsensusSpecsCheckpoint(f *testing.F) {
	fuzzConsensusSpecType[*types.Checkpoint](f, "Checkpoint")
}
func FuzzConsensusSpecsDeposit(f *testing.F) {
	fuzzConsensusSpecType[*types.Deposit](f, "Deposit")
}
func FuzzConsensusSpecsDepositData(f *testing.F) {
	fuzzConsensusSpecType[*types.DepositData](f, "DepositData")
}
func FuzzConsensusSpecsDepositMessage(f *testing.F) {
	fuzzConsensusSpecType[*types.DepositMessage](f, "DepositMessage")
}
func FuzzConsensusSpecsEth1Block(f *testing.F) {
	fuzzConsensusSpecType[*types.Eth1Block](f, "Eth1Block")
}
func FuzzConsensusSpecsEth1Data(f *testing.F) {
	fuzzConsensusSpecType[*types.Eth1Data](f, "Eth1Data")
}
func FuzzConsensusSpecsExecutionPayload(f *testing.F) {
	fuzzConsensusSpecType[*types.ExecutionPayload](f, "ExecutionPayload")
}
func FuzzConsensusSpecsExecutionPayloadCapella(f *testing.F) {
	fuzzConsensusSpecType[*types.ExecutionPayloadCapella](f, "ExecutionPayload")
}
func FuzzConsensusSpecsExecutionPayloadDeneb(f *testing.F) {
	fuzzConsensusSpecType[*types.ExecutionPayloadDeneb](f, "ExecutionPayload")
}
func FuzzConsensusSpecsExecutionPayloadHeader(f *testing.F) {
	fuzzConsensusSpecType[*types.ExecutionPayloadHeader](f, "ExecutionPayloadHeader")
}
func FuzzConsensusSpecsExecutionPayloadHeaderCapella(f *testing.F) {
	fuzzConsensusSpecType[*types.ExecutionPayloadHeaderCapella](f, "ExecutionPayloadHeader")
}
func FuzzConsensusSpecsExecutionPayloadHeaderDeneb(f *testing.F) {
	fuzzConsensusSpecType[*types.ExecutionPayloadHeaderDeneb](f, "ExecutionPayloadHeader")
}
func FuzzConsensusSpecsFork(f *testing.F) {
	fuzzConsensusSpecType[*types.Fork](f, "Fork")
}
func FuzzConsensusSpecsHistoricalBatch(f *testing.F) {
	fuzzConsensusSpecType[*types.HistoricalBatch](f, "HistoricalBatch")
}
func FuzzConsensusSpecsHistoricalSummary(f *testing.F) {
	fuzzConsensusSpecType[*types.HistoricalSummary](f, "HistoricalSummary")
}
func FuzzConsensusSpecsIndexedAttestation(f *testing.F) {
	fuzzConsensusSpecType[*types.IndexedAttestation](f, "IndexedAttestation")
}
func FuzzConsensusSpecsPendingAttestation(f *testing.F) {
	fuzzConsensusSpecType[*types.PendingAttestation](f, "PendingAttestation")
}
func FuzzConsensusSpecsProposerSlashing(f *testing.F) {
	fuzzConsensusSpecType[*types.ProposerSlashing](f, "ProposerSlashing")
}
func FuzzConsensusSpecsSignedBeaconBlockHeader(f *testing.F) {
	fuzzConsensusSpecType[*types.SignedBeaconBlockHeader](f, "SignedBeaconBlockHeader")
}
func FuzzConsensusSpecsSignedBLSToExecutionChange(f *testing.F) {
	fuzzConsensusSpecType[*types.SignedBLSToExecutionChange](f, "SignedBLSToExecutionChange")
}
func FuzzConsensusSpecsSignedVoluntaryExit(f *testing.F) {
	fuzzConsensusSpecType[*types.SignedVoluntaryExit](f, "SignedVoluntaryExit")
}
func FuzzConsensusSpecsSyncAggregate(f *testing.F) {
	fuzzConsensusSpecType[*types.SyncAggregate](f, "SyncAggregate")
}
func FuzzConsensusSpecsSyncCommittee(f *testing.F) {
	fuzzConsensusSpecType[*types.SyncCommittee](f, "SyncCommittee")
}
func FuzzConsensusSpecsValidator(f *testing.F) {
	fuzzConsensusSpecType[*types.Validator](f, "Validator")
}
func FuzzConsensusSpecsVoluntaryExit(f *testing.F) {
	fuzzConsensusSpecType[*types.VoluntaryExit](f, "VoluntaryExit")
}
func FuzzConsensusSpecsWithdrawal(f *testing.F) {
	fuzzConsensusSpecType[*types.Withdrawal](f, "Withdrawal")
}

func FuzzConsensusSpecsExecutionPayloadVariation(f *testing.F) {
	fuzzConsensusSpecType[*types.ExecutionPayloadVariation](f, "ExecutionPayload")
}
func FuzzConsensusSpecsHistoricalBatchVariation(f *testing.F) {
	fuzzConsensusSpecType[*types.HistoricalBatchVariation](f, "HistoricalBatch")
}
func FuzzConsensusSpecsWithdrawalVariation(f *testing.F) {
	fuzzConsensusSpecType[*types.WithdrawalVariation](f, "Withdrawal")
}

func fuzzConsensusSpecType[T newableObject[U], U any](f *testing.F, kind string) {
	// Iterate over all the forks and collect all the sample data
	forks, err := os.ReadDir(consensusSpecTestsRoot)
	if err != nil {
		f.Errorf("failed to walk spec collection %v: %v", consensusSpecTestsRoot, err)
		return
	}
	var valids [][]byte
	for _, fork := range forks {
		// Skip test cases for types introduced in later forks
		path := filepath.Join(consensusSpecTestsRoot, fork.Name(), "ssz_static", kind, "ssz_random")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		tests, err := os.ReadDir(path)
		if err != nil {
			f.Errorf("failed to walk test collection %v: %v", path, err)
			return
		}
		// Feed all the valid test data into the fuzzer
		for _, test := range tests {
			inSnappy, err := os.ReadFile(filepath.Join(path, test.Name(), "serialized.ssz_snappy"))
			if err != nil {
				f.Fatalf("failed to load snapy ssz binary: %v", err)
			}
			inSSZ, err := snappy.Decode(nil, inSnappy)
			if err != nil {
				f.Fatalf("failed to parse snappy ssz binary: %v", err)
			}
			obj := T(new(U))
			if err := ssz.DecodeFromStream(bytes.NewReader(inSSZ), obj, uint32(len(inSSZ))); err == nil {
				// Stash away all valid ssz streams so we can play with decoding
				// into previously used objects
				valids = append(valids, inSSZ)

				// Add the valid ssz stream to the fuzzer
				f.Add(inSSZ)
			}
		}
	}
	// Run the fuzzer
	f.Fuzz(func(t *testing.T, inSSZ []byte) {
		// Track whether the testcase is valid
		var valid bool

		// Try the stream encoder/decoder
		obj := T(new(U))
		if err := ssz.DecodeFromStream(bytes.NewReader(inSSZ), obj, uint32(len(inSSZ))); err == nil {
			// Stream decoder succeeded, make sure it re-encodes correctly and
			// that the buffer decoder also succeeds parsing
			blob := new(bytes.Buffer)
			if err := ssz.EncodeToStream(blob, obj); err != nil {
				t.Fatalf("failed to re-encode stream: %v", err)
			}
			if !bytes.Equal(blob.Bytes(), inSSZ) {
				prefix := commonPrefix(blob.Bytes(), inSSZ)
				t.Fatalf("re-encoded stream mismatch: have %x, want %x, common prefix %d, have left %x, want left %x",
					blob, inSSZ, len(prefix), blob.Bytes()[len(prefix):], inSSZ[len(prefix):])
			}
			if err := ssz.DecodeFromBytes(inSSZ, obj); err != nil {
				t.Fatalf("failed to decode buffer: %v", err)
			}
			// Sanity check that hashing and size retrieval works
			hash1 := ssz.HashSequential(obj)
			hash2 := ssz.HashConcurrent(obj)
			if hash1 != hash2 {
				t.Fatalf("sequential/concurrent hash mismatch: sequencial %x, concurrent %x", hash1, hash2)
			}
			if size := ssz.Size(obj); size != uint32(len(inSSZ)) {
				t.Fatalf("reported/generated size mismatch: reported %v, generated %v", size, len(inSSZ))
			}
			valid = true
		}
		// Try the buffer encoder/decoder
		obj = T(new(U))
		if err := ssz.DecodeFromBytes(inSSZ, obj); err == nil {
			// Buffer decoder succeeded, make sure it re-encodes correctly and
			// that the stream decoder also succeeds parsing
			bin := make([]byte, ssz.Size(obj))
			if err := ssz.EncodeToBytes(bin, obj); err != nil {
				t.Fatalf("failed to re-encode buffer: %v", err)
			}
			if !bytes.Equal(bin, inSSZ) {
				prefix := commonPrefix(bin, inSSZ)
				t.Fatalf("re-encoded buffer mismatch: have %x, want %x, common prefix %d, have left %x, want left %x",
					bin, inSSZ, len(prefix), bin[len(prefix):], inSSZ[len(prefix):])
			}
			if err := ssz.DecodeFromStream(bytes.NewReader(inSSZ), obj, uint32(len(inSSZ))); err != nil {
				t.Fatalf("failed to decode stream: %v", err)
			}
			// Sanity check that hashing and size retrieval works
			hash1 := ssz.HashSequential(obj)
			hash2 := ssz.HashConcurrent(obj)
			if hash1 != hash2 {
				t.Fatalf("sequential/concurrent hash mismatch: sequencial %x, concurrent %x", hash1, hash2)
			}
			if size := ssz.Size(obj); size != uint32(len(inSSZ)) {
				t.Fatalf("reported/generated size mismatch: reported %v, generated %v", size, len(inSSZ))
			}
		}
		// If the testcase was valid, try decoding it into a used object
		if valid {
			// Pick a random starting object
			vSSZ := valids[rand.Intn(len(valids))]

			// Try the stream encoder/decoder into a prepped object
			obj = T(new(U))
			if err := ssz.DecodeFromBytes(vSSZ, obj); err != nil {
				panic(err) // we've already decoded this, cannot fail
			}
			if err := ssz.DecodeFromStream(bytes.NewReader(inSSZ), obj, uint32(len(inSSZ))); err != nil {
				t.Fatalf("failed to decode stream into used object: %v", err)
			}
			blob := new(bytes.Buffer)
			if err := ssz.EncodeToStream(blob, obj); err != nil {
				t.Fatalf("failed to re-encode stream from used object: %v", err)
			}
			if !bytes.Equal(blob.Bytes(), inSSZ) {
				prefix := commonPrefix(blob.Bytes(), inSSZ)
				t.Fatalf("re-encoded stream from used object mismatch: have %x, want %x, common prefix %d, have left %x, want left %x",
					blob, inSSZ, len(prefix), blob.Bytes()[len(prefix):], inSSZ[len(prefix):])
			}
			hash1 := ssz.HashSequential(obj)
			hash2 := ssz.HashConcurrent(obj)
			if hash1 != hash2 {
				t.Fatalf("sequential/concurrent hash mismatch: sequencial %x, concurrent %x", hash1, hash2)
			}
			if size := ssz.Size(obj); size != uint32(len(inSSZ)) {
				t.Fatalf("reported/generated size mismatch: reported %v, generated %v", size, len(inSSZ))
			}
			// Try the buffer encoder/decoder into a prepped object
			obj = T(new(U))
			if err := ssz.DecodeFromBytes(vSSZ, obj); err != nil {
				panic(err) // we've already decoded this, cannot fail
			}
			if err := ssz.DecodeFromBytes(inSSZ, obj); err != nil {
				t.Fatalf("failed to decode buffer into used object: %v", err)
			}
			bin := make([]byte, ssz.Size(obj))
			if err := ssz.EncodeToBytes(bin, obj); err != nil {
				t.Fatalf("failed to re-encode buffer from used object: %v", err)
			}
			if !bytes.Equal(bin, inSSZ) {
				prefix := commonPrefix(bin, inSSZ)
				t.Fatalf("re-encoded buffer from used object mismatch: have %x, want %x, common prefix %d, have left %x, want left %x",
					blob, inSSZ, len(prefix), bin[len(prefix):], inSSZ[len(prefix):])
			}
			hash1 = ssz.HashSequential(obj)
			hash2 = ssz.HashConcurrent(obj)
			if hash1 != hash2 {
				t.Fatalf("sequential/concurrent hash mismatch: sequencial %x, concurrent %x", hash1, hash2)
			}
			if size := ssz.Size(obj); size != uint32(len(inSSZ)) {
				t.Fatalf("reported/generated size mismatch: reported %v, generated %v", size, len(inSSZ))
			}
		}
	})
}
