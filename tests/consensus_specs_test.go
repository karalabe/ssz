// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package tests

import (
	"bytes"
	"fmt"
	"io"
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

// TestConsensusSpecs iterates over all the (supported) consensus SSZ types and
// runs the encoding/decoding/hashing round.
func TestConsensusSpecs(t *testing.T) {
	testConsensusSpecType[*types.Attestation](t, "Attestation", "altair", "bellatrix", "capella", "deneb", "eip7594", "phase0", "whisk")
	testConsensusSpecType[*types.AttestationData](t, "AttestationData")
	testConsensusSpecType[*types.AttesterSlashing](t, "AttesterSlashing")
	testConsensusSpecType[*types.BeaconBlock](t, "BeaconBlock", "phase0")
	testConsensusSpecType[*types.BeaconBlockBody](t, "BeaconBlockBody", "phase0")
	testConsensusSpecType[*types.BeaconBlockHeader](t, "BeaconBlockHeader")
	testConsensusSpecType[*types.Checkpoint](t, "Checkpoint")
	testConsensusSpecType[*types.Deposit](t, "Deposit")
	testConsensusSpecType[*types.DepositData](t, "DepositData")
	testConsensusSpecType[*types.Eth1Data](t, "Eth1Data")
	testConsensusSpecType[*types.ExecutionPayload](t, "bellatrix", "ExecutionPayload")
	testConsensusSpecType[*types.ExecutionPayloadCapella](t, "capella", "ExecutionPayload")
	testConsensusSpecType[*types.HistoricalBatch](t, "HistoricalBatch")
	testConsensusSpecType[*types.IndexedAttestation](t, "IndexedAttestation")
	testConsensusSpecType[*types.ProposerSlashing](t, "ProposerSlashing")
	testConsensusSpecType[*types.SignedBeaconBlockHeader](t, "SignedBeaconBlockHeader")
	testConsensusSpecType[*types.SignedVoluntaryExit](t, "SignedVoluntaryExit")
	testConsensusSpecType[*types.VoluntaryExit](t, "VoluntaryExit")
	testConsensusSpecType[*types.Withdrawal](t, "Withdrawal")

	// Iterate over all the untouched tests and report them
	/*forks, err := os.ReadDir(consensusSpecTestsRoot)
	if err != nil {
		t.Fatalf("failed to walk fork collection: %v", err)
	}
	for _, fork := range forks {
		if _, ok := consensusSpecTestsDone[fork.Name()]; !ok {
			t.Errorf("no tests ran for %v", fork.Name())
			continue
		}
		types, err := os.ReadDir(filepath.Join(consensusSpecTestsRoot, fork.Name(), "ssz_static"))
		if err != nil {
			t.Fatalf("failed to walk type collection of %v: %v", fork, err)
		}
		for _, kind := range types {
			if _, ok := consensusSpecTestsDone[fork.Name()][kind.Name()]; !ok {
				t.Errorf("no tests ran for %v/%v", fork.Name(), kind.Name())
			}
		}
	}*/
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
					t.Fatalf("re-encoded stream mismatch: have %x, want %x", blob, inSSZ)
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
					t.Fatalf("re-encoded bytes mismatch: have %x, want %x", bin, inSSZ)
				}
				// Encoder/decoder seems to work, check if the size reported by the
				// encoded object actually matches the encoded stream
				if size := ssz.Size(obj); size != uint32(len(inSSZ)) {
					t.Fatalf("reported/generated size mismatch: reported %v, generated %v", size, len(inSSZ))
				}
				// TODO(karalabe): check the root hash of the object
			})
		}
	}
}

// TestConsensusSpecs iterates over all the (supported) consensus SSZ types and
// runs the encoding/decoding/hashing benchmark round.
func BenchmarkConsensusSpecs(b *testing.B) {
	benchmarkConsensusSpecType[*types.Attestation](b, "deneb", "Attestation")
	benchmarkConsensusSpecType[*types.AttestationData](b, "deneb", "AttestationData")
	benchmarkConsensusSpecType[*types.AttesterSlashing](b, "deneb", "AttesterSlashing")
	benchmarkConsensusSpecType[*types.BeaconBlock](b, "phase0", "BeaconBlock")
	benchmarkConsensusSpecType[*types.BeaconBlockBody](b, "phase0", "BeaconBlockBody")
	benchmarkConsensusSpecType[*types.BeaconBlockHeader](b, "deneb", "BeaconBlockHeader")
	benchmarkConsensusSpecType[*types.Checkpoint](b, "deneb", "Checkpoint")
	benchmarkConsensusSpecType[*types.Deposit](b, "deneb", "Deposit")
	benchmarkConsensusSpecType[*types.DepositData](b, "deneb", "DepositData")
	benchmarkConsensusSpecType[*types.Eth1Data](b, "deneb", "Eth1Data")
	benchmarkConsensusSpecType[*types.ExecutionPayloadCapella](b, "capella", "ExecutionPayload")
	benchmarkConsensusSpecType[*types.HistoricalBatch](b, "deneb", "HistoricalBatch")
	benchmarkConsensusSpecType[*types.IndexedAttestation](b, "deneb", "IndexedAttestation")
	benchmarkConsensusSpecType[*types.ProposerSlashing](b, "deneb", "ProposerSlashing")
	benchmarkConsensusSpecType[*types.SignedBeaconBlockHeader](b, "deneb", "SignedBeaconBlockHeader")
	benchmarkConsensusSpecType[*types.SignedVoluntaryExit](b, "deneb", "SignedVoluntaryExit")
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
}
