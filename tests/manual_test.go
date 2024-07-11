package tests

import (
	"fmt"
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
	fmt.Println(obj.Slot)

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
			ssz.MerkleizeSequential(obj)
			//fmt.Printf("%#x\n", ssz.MerkleizeSequential(obj))
		}
	})
	b.Run(fmt.Sprintf("beacon-state/%d-bytes/merkleize-concurrent", len(blob)), func(b *testing.B) {
		b.SetBytes(int64(len(blob)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ssz.MerkleizeConcurrent(obj)
			//fmt.Printf("%#x\n", ssz.MerkleizeConcurrent(obj))
		}
	})
}
