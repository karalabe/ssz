// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package tests

import (
	"bytes"
	"fmt"
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
	// Run the single-version type tests
	testConsensusSpecType[*types.AttestationData](t, "", "AttestationData")
	testConsensusSpecType[*types.AttesterSlashing](t, "", "AttesterSlashing")
	testConsensusSpecType[*types.BeaconBlockHeader](t, "", "BeaconBlockHeader")
	testConsensusSpecType[*types.Checkpoint](t, "", "Checkpoint")
	testConsensusSpecType[*types.HistoricalBatch](t, "", "HistoricalBatch")
	testConsensusSpecType[*types.IndexedAttestation](t, "", "IndexedAttestation")
	testConsensusSpecType[*types.ProposerSlashing](t, "", "ProposerSlashing")
	testConsensusSpecType[*types.SignedBeaconBlockHeader](t, "", "SignedBeaconBlockHeader")
	testConsensusSpecType[*types.Withdrawal](t, "", "Withdrawal")

	// Run the fork-specific type tests
	testConsensusSpecType[*types.ExecutionPayloadBellatrix](t, "bellatrix", "ExecutionPayload")
	testConsensusSpecType[*types.ExecutionPayloadCapella](t, "capella", "ExecutionPayload")

	// Iterate over all the untouched tests and report them
	forks, err := os.ReadDir(consensusSpecTestsRoot)
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
	}
}

// newableObject is a generic type whose purpose is to enforce that ssz.Object
// is specifically implemented on a struct pointer. That's needed to allow us
// to instantiate new structs via `new` when parsing.
type newableObject[U any] interface {
	ssz.Object
	*U
}

func testConsensusSpecType[T newableObject[U], U any](t *testing.T, fork, kind string) {
	// If no fork was specified, iterate over all of them and use the same type
	if fork == "" {
		forks, err := os.ReadDir(consensusSpecTestsRoot)
		if err != nil {
			t.Errorf("failed to walk spec collection %v: %v", consensusSpecTestsRoot, err)
			return
		}
		for _, fork := range forks {
			if _, err := os.Stat(filepath.Join(consensusSpecTestsRoot, fork.Name(), "ssz_static", kind, "ssz_random")); err == nil {
				testConsensusSpecType[T, U](t, fork.Name(), kind)
			}
		}
		return
	}
	// Some specific fork was requested, look that up explicitly
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
			if err := ssz.Decode(bytes.NewReader(inSSZ), obj, uint32(len(inSSZ))); err != nil {
				t.Fatalf("failed to decode SSZ stream: %v", err)
			}
			blob := new(bytes.Buffer)
			if err := ssz.Encode(blob, obj); err != nil {
				t.Fatalf("failed to re-encode SSZ stream: %v", err)
			}
			if !bytes.Equal(blob.Bytes(), inSSZ) {
				t.Fatalf("re-encoded stream mismatch: have %x, want %x", blob, inSSZ)
			}
			// Encoder/decoder seems to work, check if the size reported by the
			// encoded object actually matches the encoded stream
			if size := obj.SizeSSZ(); size != uint32(len(inSSZ)) {
				t.Fatalf("reported/generated size mismatch: reported %v, generated %v", size, len(inSSZ))
			}
			// TODO(karalabe): check the root hash of the object
		})
	}
}

// TestConsensusSpecs iterates over all the (supported) consensus SSZ types and
// runs the encoding/decoding/hashing benchmark round.
func BenchmarkConsensusSpecs(b *testing.B) {
	benchmarkConsensusSpecType[*types.AttestationData](b, "deneb", "AttestationData", "case_4")
	benchmarkConsensusSpecType[*types.AttesterSlashing](b, "deneb", "AttesterSlashing", "case_4")
	benchmarkConsensusSpecType[*types.BeaconBlockHeader](b, "deneb", "BeaconBlockHeader", "case_4")
	benchmarkConsensusSpecType[*types.Checkpoint](b, "deneb", "Checkpoint", "case_4")
	benchmarkConsensusSpecType[*types.ExecutionPayloadCapella](b, "capella", "ExecutionPayload", "case_4")
	benchmarkConsensusSpecType[*types.HistoricalBatch](b, "deneb", "HistoricalBatch", "case_4")
	benchmarkConsensusSpecType[*types.IndexedAttestation](b, "deneb", "IndexedAttestation", "case_4")
	benchmarkConsensusSpecType[*types.ProposerSlashing](b, "deneb", "ProposerSlashing", "case_4")
	benchmarkConsensusSpecType[*types.SignedBeaconBlockHeader](b, "deneb", "SignedBeaconBlockHeader", "case_4")
	benchmarkConsensusSpecType[*types.Withdrawal](b, "deneb", "Withdrawal", "case_4")
}

func benchmarkConsensusSpecType[T newableObject[U], U any](b *testing.B, fork, kind, test string) {
	path := filepath.Join(consensusSpecTestsRoot, fork, "ssz_static", kind, "ssz_random", test)

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
	if err := ssz.Decode(bytes.NewReader(inSSZ), inObj, uint32(len(inSSZ))); err != nil {
		b.Fatalf("failed to decode SSZ stream: %v", err)
	}
	// Start the benchmarks for all the different operations
	b.Run(fmt.Sprintf("%s/encode", kind), func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(inSSZ)))

		blob := make([]byte, inObj.SizeSSZ())
		for i := 0; i < b.N; i++ {
			if err := ssz.Encode(bytes.NewBuffer(blob[:0]), inObj); err != nil {
				b.Fatalf("failed to re-encode SSZ stream: %v", err)
			}
		}
	})
	b.Run(fmt.Sprintf("%s/decode", kind), func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(inSSZ)))

		obj := T(new(U))
		for i := 0; i < b.N; i++ {
			if err := ssz.Decode(bytes.NewReader(inSSZ), obj, uint32(len(inSSZ))); err != nil {
				b.Fatalf("failed to decode SSZ stream: %v", err)
			}
		}
	})
	b.Run(fmt.Sprintf("%s/size", kind), func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			inObj.SizeSSZ()
		}
	})
}
