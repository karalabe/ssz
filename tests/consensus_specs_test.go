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
	"strings"
	"sync"
	"testing"

	"github.com/golang/snappy"
	"github.com/karalabe/ssz"
	types "github.com/karalabe/ssz/tests/testtypes/consensus-spec-tests"
	"gopkg.in/yaml.v3"
)

var (
	// consensusSpecTestsBasicsRoot is the folder where the basic ssz tests are located.
	consensusSpecTestsBasicsRoot = filepath.Join("testdata", "consensus-spec-tests", "tests", "general", "phase0", "ssz_generic", "containers")

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

// TestConsensusSpecBasics iterates over the basic container tests from the
// consensus spec tests repo and runs the encoding/decoding/hashing round.
func TestConsensusSpecBasics(t *testing.T) {
	testConsensusSpecBasicType[*ssz.Codec, *types.SingleFieldTestStruct, types.SingleFieldTestStruct](t, "SingleFieldTestStruct")
	testConsensusSpecBasicType[*ssz.Codec, *types.SmallTestStruct, types.SmallTestStruct](t, "SmallTestStruct")
	testConsensusSpecBasicType[*ssz.Codec, *types.FixedTestStruct, types.FixedTestStruct](t, "FixedTestStruct")
	testConsensusSpecBasicType[*ssz.Codec, *types.BitsStruct, types.BitsStruct](t, "BitsStruct")
}

func testConsensusSpecBasicType[C ssz.CodecI[C], T newableObject[C, U], U any](t *testing.T, kind string) {
	// Filter out the valid tests for this specific type
	path := filepath.Join(consensusSpecTestsBasicsRoot, "valid")

	tests, err := os.ReadDir(path)
	if err != nil {
		t.Errorf("failed to walk valid test collection %v: %v", path, err)
		return
	}
	for i := 0; i < len(tests); i++ {
		if !strings.HasPrefix(tests[i].Name(), kind+"_") {
			tests = append(tests[:i], tests[i+1:]...)
			i--
		}
	}
	// Run all the valid tests
	for _, test := range tests {
		t.Run(fmt.Sprintf("valid/%s/%s", kind, test.Name()), func(t *testing.T) {
			// Parse the input SSZ data and the expected root for the test
			inSnappy, err := os.ReadFile(filepath.Join(path, test.Name(), "serialized.ssz_snappy"))
			if err != nil {
				t.Fatalf("failed to load snapy ssz binary: %v", err)
			}
			inSSZ, err := snappy.Decode(nil, inSnappy)
			if err != nil {
				t.Fatalf("failed to parse snappy ssz binary: %v", err)
			}
			inYAML, err := os.ReadFile(filepath.Join(path, test.Name(), "meta.yaml"))
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
	// Filter out the valid tests for this specific type
	path = filepath.Join(consensusSpecTestsBasicsRoot, "invalid")

	tests, err = os.ReadDir(path)
	if err != nil {
		t.Errorf("failed to walk invalid test collection %v: %v", path, err)
		return
	}
	for i := 0; i < len(tests); i++ {
		if !strings.HasPrefix(tests[i].Name(), kind+"_") {
			tests = append(tests[:i], tests[i+1:]...)
			i--
		}
	}
	// Run all the valid tests
	for _, test := range tests {
		t.Run(fmt.Sprintf("invalid/%s/%s", kind, test.Name()), func(t *testing.T) {
			// Parse the input SSZ data and the expected root for the test
			inSnappy, err := os.ReadFile(filepath.Join(path, test.Name(), "serialized.ssz_snappy"))
			if err != nil {
				t.Fatalf("failed to load snapy ssz binary: %v", err)
			}
			inSSZ, err := snappy.Decode(nil, inSnappy)
			if err != nil {
				t.Fatalf("failed to parse snappy ssz binary: %v", err)
			}
			// Try to decode, it should fail
			obj := T(new(U))
			if err := ssz.DecodeFromStream(bytes.NewReader(inSSZ), obj, uint32(len(inSSZ))); err == nil {
				t.Fatalf("succeeded in decoding invalid SSZ stream")
			}
			obj = T(new(U))
			if err := ssz.DecodeFromBytes(inSSZ, obj); err == nil {
				t.Fatalf("succeeded in decoding invalid SSZ buffer")
			}
		})
	}
}

// TestConsensusSpecs iterates over all the (supported) consensus SSZ types and
// runs the encoding/decoding/hashing round.
func TestConsensusSpecs(t *testing.T) {
	testConsensusSpecType[*ssz.Codec, *types.AggregateAndProof](t, "AggregateAndProof", "altair", "bellatrix", "capella", "deneb", "eip7594", "phase0", "whisk")
	testConsensusSpecType[*ssz.Codec, *types.Attestation](t, "Attestation", "altair", "bellatrix", "capella", "deneb", "eip7594", "phase0", "whisk")
	testConsensusSpecType[*ssz.Codec, *types.AttestationData](t, "AttestationData")
	testConsensusSpecType[*ssz.Codec, *types.AttesterSlashing](t, "AttesterSlashing", "phase0", "altair", "bellatrix", "capella", "deneb")
	testConsensusSpecType[*ssz.Codec, *types.BeaconBlock](t, "BeaconBlock", "phase0")
	testConsensusSpecType[*ssz.Codec, *types.BeaconBlockBody](t, "BeaconBlockBody", "phase0")
	testConsensusSpecType[*ssz.Codec, *types.BeaconBlockBodyAltair](t, "BeaconBlockBody", "altair")
	testConsensusSpecType[*ssz.Codec, *types.BeaconBlockBodyBellatrix](t, "BeaconBlockBody", "bellatrix")
	testConsensusSpecType[*ssz.Codec, *types.BeaconBlockBodyCapella](t, "BeaconBlockBody", "capella")
	testConsensusSpecType[*ssz.Codec, *types.BeaconBlockBodyDeneb](t, "BeaconBlockBody", "deneb", "eip7594")
	testConsensusSpecType[*ssz.Codec, *types.BeaconBlockHeader](t, "BeaconBlockHeader")
	testConsensusSpecType[*ssz.Codec, *types.BeaconState](t, "BeaconState", "phase0")
	testConsensusSpecType[*ssz.Codec, *types.BeaconStateCapella](t, "BeaconState", "capella")
	testConsensusSpecType[*ssz.Codec, *types.BeaconStateDeneb](t, "BeaconState", "deneb")
	testConsensusSpecType[*ssz.Codec, *types.BLSToExecutionChange](t, "BLSToExecutionChange")
	testConsensusSpecType[*ssz.Codec, *types.Checkpoint](t, "Checkpoint")
	testConsensusSpecType[*ssz.Codec, *types.Deposit](t, "Deposit")
	testConsensusSpecType[*ssz.Codec, *types.DepositData](t, "DepositData")
	testConsensusSpecType[*ssz.Codec, *types.DepositMessage](t, "DepositMessage")
	testConsensusSpecType[*ssz.Codec, *types.Eth1Block](t, "Eth1Block")
	testConsensusSpecType[*ssz.Codec, *types.Eth1Data](t, "Eth1Data")
	testConsensusSpecType[*ssz.Codec, *types.ExecutionPayload](t, "ExecutionPayload", "bellatrix")
	testConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadHeader](t, "ExecutionPayloadHeader", "bellatrix")
	testConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadCapella](t, "ExecutionPayload", "capella")
	testConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadHeaderCapella](t, "ExecutionPayloadHeader", "capella")
	testConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadDeneb](t, "ExecutionPayload", "deneb", "eip7594")
	testConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadHeaderDeneb](t, "ExecutionPayloadHeader", "deneb", "eip7594")
	testConsensusSpecType[*ssz.Codec, *types.Fork](t, "Fork")
	testConsensusSpecType[*ssz.Codec, *types.HistoricalBatch](t, "HistoricalBatch")
	testConsensusSpecType[*ssz.Codec, *types.HistoricalSummary](t, "HistoricalSummary")
	testConsensusSpecType[*ssz.Codec, *types.IndexedAttestation](t, "IndexedAttestation", "phase0", "altair", "bellatrix", "capella", "deneb")
	testConsensusSpecType[*ssz.Codec, *types.PendingAttestation](t, "PendingAttestation")
	testConsensusSpecType[*ssz.Codec, *types.ProposerSlashing](t, "ProposerSlashing")
	testConsensusSpecType[*ssz.Codec, *types.SignedBeaconBlockHeader](t, "SignedBeaconBlockHeader")
	testConsensusSpecType[*ssz.Codec, *types.SignedBLSToExecutionChange](t, "SignedBLSToExecutionChange")
	testConsensusSpecType[*ssz.Codec, *types.SignedVoluntaryExit](t, "SignedVoluntaryExit")
	testConsensusSpecType[*ssz.Codec, *types.SyncAggregate](t, "SyncAggregate")
	testConsensusSpecType[*ssz.Codec, *types.SyncCommittee](t, "SyncCommittee")
	testConsensusSpecType[*ssz.Codec, *types.Validator](t, "Validator")
	testConsensusSpecType[*ssz.Codec, *types.VoluntaryExit](t, "VoluntaryExit")
	testConsensusSpecType[*ssz.Codec, *types.Withdrawal](t, "Withdrawal")

	// Add some API variations to test different codec implementations
	testConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadVariation](t, "ExecutionPayload", "bellatrix")
	testConsensusSpecType[*ssz.Codec, *types.HistoricalBatchVariation](t, "HistoricalBatch")
	testConsensusSpecType[*ssz.Codec, *types.WithdrawalVariation](t, "Withdrawal")

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
type newableObject[C ssz.CodecI[C], U any] interface {
	ssz.Object[C]
	*U
}

func testConsensusSpecType[C ssz.CodecI[C], T newableObject[C, U], U any](t *testing.T, kind string, forks ...string) {
	// If no fork was specified, iterate over all of them and use the same type
	if len(forks) == 0 {
		forks, err := os.ReadDir(consensusSpecTestsRoot)
		if err != nil {
			t.Errorf("failed to walk spec collection %v: %v", consensusSpecTestsRoot, err)
			return
		}
		for _, fork := range forks {
			if _, err := os.Stat(filepath.Join(consensusSpecTestsRoot, fork.Name(), "ssz_static", kind, "ssz_random")); err == nil {
				testConsensusSpecType[C, T, U](t, kind, fork.Name())
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
	benchmarkConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadVariation](b, "bellatrix", "ExecutionPayload")

	benchmarkConsensusSpecType[*ssz.Codec, *types.AggregateAndProof](b, "deneb", "AggregateAndProof")
	benchmarkConsensusSpecType[*ssz.Codec, *types.Attestation](b, "deneb", "Attestation")
	benchmarkConsensusSpecType[*ssz.Codec, *types.AttestationData](b, "deneb", "AttestationData")
	benchmarkConsensusSpecType[*ssz.Codec, *types.AttesterSlashing](b, "deneb", "AttesterSlashing")
	benchmarkConsensusSpecType[*ssz.Codec, *types.BeaconBlock](b, "phase0", "BeaconBlock")
	benchmarkConsensusSpecType[*ssz.Codec, *types.BeaconBlockBodyDeneb](b, "deneb", "BeaconBlockBody")
	benchmarkConsensusSpecType[*ssz.Codec, *types.BeaconBlockHeader](b, "deneb", "BeaconBlockHeader")
	benchmarkConsensusSpecType[*ssz.Codec, *types.BeaconState](b, "phase0", "BeaconState")
	benchmarkConsensusSpecType[*ssz.Codec, *types.BLSToExecutionChange](b, "deneb", "BLSToExecutionChange")
	benchmarkConsensusSpecType[*ssz.Codec, *types.Checkpoint](b, "deneb", "Checkpoint")
	benchmarkConsensusSpecType[*ssz.Codec, *types.Deposit](b, "deneb", "Deposit")
	benchmarkConsensusSpecType[*ssz.Codec, *types.DepositData](b, "deneb", "DepositData")
	benchmarkConsensusSpecType[*ssz.Codec, *types.DepositMessage](b, "deneb", "DepositMessage")
	benchmarkConsensusSpecType[*ssz.Codec, *types.Eth1Block](b, "deneb", "Eth1Block")
	benchmarkConsensusSpecType[*ssz.Codec, *types.Eth1Data](b, "deneb", "Eth1Data")
	benchmarkConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadDeneb](b, "deneb", "ExecutionPayload")
	benchmarkConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadHeaderDeneb](b, "deneb", "ExecutionPayloadHeader")
	benchmarkConsensusSpecType[*ssz.Codec, *types.Fork](b, "deneb", "Fork")
	benchmarkConsensusSpecType[*ssz.Codec, *types.HistoricalBatch](b, "deneb", "HistoricalBatch")
	benchmarkConsensusSpecType[*ssz.Codec, *types.HistoricalSummary](b, "deneb", "HistoricalSummary")
	benchmarkConsensusSpecType[*ssz.Codec, *types.IndexedAttestation](b, "deneb", "IndexedAttestation")
	benchmarkConsensusSpecType[*ssz.Codec, *types.PendingAttestation](b, "deneb", "PendingAttestation")
	benchmarkConsensusSpecType[*ssz.Codec, *types.ProposerSlashing](b, "deneb", "ProposerSlashing")
	benchmarkConsensusSpecType[*ssz.Codec, *types.SignedBeaconBlockHeader](b, "deneb", "SignedBeaconBlockHeader")
	benchmarkConsensusSpecType[*ssz.Codec, *types.SignedBLSToExecutionChange](b, "deneb", "SignedBLSToExecutionChange")
	benchmarkConsensusSpecType[*ssz.Codec, *types.SignedVoluntaryExit](b, "deneb", "SignedVoluntaryExit")
	benchmarkConsensusSpecType[*ssz.Codec, *types.SyncAggregate](b, "deneb", "SyncAggregate")
	benchmarkConsensusSpecType[*ssz.Codec, *types.SyncCommittee](b, "deneb", "SyncCommittee")
	benchmarkConsensusSpecType[*ssz.Codec, *types.Validator](b, "deneb", "Validator")
	benchmarkConsensusSpecType[*ssz.Codec, *types.VoluntaryExit](b, "deneb", "VoluntaryExit")
	benchmarkConsensusSpecType[*ssz.Codec, *types.Withdrawal](b, "deneb", "Withdrawal")
}

func benchmarkConsensusSpecType[C ssz.CodecI[C], T newableObject[C, U], U any](b *testing.B, fork, kind string) {
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
	fuzzConsensusSpecType[*ssz.Codec, *types.AggregateAndProof](f, "AggregateAndProof")
}
func FuzzConsensusSpecsAttestation(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.Attestation](f, "Attestation")
}
func FuzzConsensusSpecsAttestationData(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.AttestationData](f, "AttestationData")
}
func FuzzConsensusSpecsAttesterSlashing(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.AttesterSlashing](f, "AttesterSlashing")
}
func FuzzConsensusSpecsBeaconBlock(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.BeaconBlock](f, "BeaconBlock")
}
func FuzzConsensusSpecsBeaconBlockBody(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.BeaconBlockBody](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockBodyAltair(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.BeaconBlockBodyAltair](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockBodyBellatrix(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.BeaconBlockBodyBellatrix](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockBodyCapella(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.BeaconBlockBodyCapella](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockBodyDeneb(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.BeaconBlockBodyDeneb](f, "BeaconBlockBody")
}
func FuzzConsensusSpecsBeaconBlockHeader(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.BeaconBlockHeader](f, "BeaconBlockHeader")
}
func FuzzConsensusSpecsBeaconState(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.BeaconState](f, "BeaconState")
}
func FuzzConsensusSpecsBLSToExecutionChange(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.BLSToExecutionChange](f, "BLSToExecutionChange")
}
func FuzzConsensusSpecsCheckpoint(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.Checkpoint](f, "Checkpoint")
}
func FuzzConsensusSpecsDeposit(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.Deposit](f, "Deposit")
}
func FuzzConsensusSpecsDepositData(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.DepositData](f, "DepositData")
}
func FuzzConsensusSpecsDepositMessage(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.DepositMessage](f, "DepositMessage")
}
func FuzzConsensusSpecsEth1Block(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.Eth1Block](f, "Eth1Block")
}
func FuzzConsensusSpecsEth1Data(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.Eth1Data](f, "Eth1Data")
}
func FuzzConsensusSpecsExecutionPayload(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.ExecutionPayload](f, "ExecutionPayload")
}
func FuzzConsensusSpecsExecutionPayloadCapella(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadCapella](f, "ExecutionPayload")
}
func FuzzConsensusSpecsExecutionPayloadDeneb(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadDeneb](f, "ExecutionPayload")
}
func FuzzConsensusSpecsExecutionPayloadHeader(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadHeader](f, "ExecutionPayloadHeader")
}
func FuzzConsensusSpecsExecutionPayloadHeaderCapella(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadHeaderCapella](f, "ExecutionPayloadHeader")
}
func FuzzConsensusSpecsExecutionPayloadHeaderDeneb(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadHeaderDeneb](f, "ExecutionPayloadHeader")
}
func FuzzConsensusSpecsFork(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.Fork](f, "Fork")
}
func FuzzConsensusSpecsHistoricalBatch(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.HistoricalBatch](f, "HistoricalBatch")
}
func FuzzConsensusSpecsHistoricalSummary(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.HistoricalSummary](f, "HistoricalSummary")
}
func FuzzConsensusSpecsIndexedAttestation(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.IndexedAttestation](f, "IndexedAttestation")
}
func FuzzConsensusSpecsPendingAttestation(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.PendingAttestation](f, "PendingAttestation")
}
func FuzzConsensusSpecsProposerSlashing(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.ProposerSlashing](f, "ProposerSlashing")
}
func FuzzConsensusSpecsSignedBeaconBlockHeader(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.SignedBeaconBlockHeader](f, "SignedBeaconBlockHeader")
}
func FuzzConsensusSpecsSignedBLSToExecutionChange(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.SignedBLSToExecutionChange](f, "SignedBLSToExecutionChange")
}
func FuzzConsensusSpecsSignedVoluntaryExit(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.SignedVoluntaryExit](f, "SignedVoluntaryExit")
}
func FuzzConsensusSpecsSyncAggregate(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.SyncAggregate](f, "SyncAggregate")
}
func FuzzConsensusSpecsSyncCommittee(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.SyncCommittee](f, "SyncCommittee")
}
func FuzzConsensusSpecsValidator(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.Validator](f, "Validator")
}
func FuzzConsensusSpecsVoluntaryExit(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.VoluntaryExit](f, "VoluntaryExit")
}
func FuzzConsensusSpecsWithdrawal(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.Withdrawal](f, "Withdrawal")
}

func FuzzConsensusSpecsExecutionPayloadVariation(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.ExecutionPayloadVariation](f, "ExecutionPayload")
}
func FuzzConsensusSpecsHistoricalBatchVariation(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.HistoricalBatchVariation](f, "HistoricalBatch")
}
func FuzzConsensusSpecsWithdrawalVariation(f *testing.F) {
	fuzzConsensusSpecType[*ssz.Codec, *types.WithdrawalVariation](f, "Withdrawal")
}

func fuzzConsensusSpecType[C ssz.CodecI[C], T newableObject[C, U], U any](f *testing.F, kind string) {
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
