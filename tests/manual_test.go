//go:build ignore

package tests

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/karalabe/ssz"
	types "github.com/karalabe/ssz/tests/testtypes/consensus-spec-tests"
)

func BenchmarkMainnetState(b *testing.B) {
	blob, err := os.ReadFile("../state.ssz")
	if err != nil {
		panic(err)
	}
	obj := new(types.BeaconStateDeneb)
	if err := ssz.DecodeFromBytes(blob, obj); err != nil {
		panic(err)
	}
	hash := ssz.MerkleizeSequential(obj)

	b.Run(fmt.Sprintf("beacon-state/%d-bytes/encode", len(blob)), func(b *testing.B) {
		b.SetBytes(int64(len(blob)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ssz.EncodeToStream(io.Discard, obj)
		}
	})
	b.Run(fmt.Sprintf("beacon-state/%d-bytes/decode", len(blob)), func(b *testing.B) {
		obj := new(types.BeaconStateDeneb)

		b.SetBytes(int64(len(blob)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ssz.DecodeFromBytes(blob, obj)
		}
	})
	b.Run(fmt.Sprintf("beacon-state/%d-bytes/merkleize-sequential", len(blob)), func(b *testing.B) {
		b.SetBytes(int64(len(blob)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if ssz.MerkleizeSequential(obj) != hash {
				panic("hash mismatch")
			}
		}
	})
	b.Run(fmt.Sprintf("beacon-state/%d-bytes/merkleize-concurrent", len(blob)), func(b *testing.B) {
		b.SetBytes(int64(len(blob)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if ssz.MerkleizeConcurrent(obj) != hash {
				panic("hash mismatch")
			}
		}
	})
}
