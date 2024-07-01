// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import (
	"github.com/karalabe/ssz"
)

type HistoricalBatch struct {
	BlockRoots [8192]Hash
	StateRoots [8192]Hash
}

func (h *HistoricalBatch) StaticSSZ() bool { return true }
func (h *HistoricalBatch) SizeSSZ() uint32 { return 2 * 8192 * 32 }

func (h *HistoricalBatch) EncodeSSZ(enc *ssz.Encoder) {
	// Initialize a dynamic encoder with the given starting offsets
	defer enc.OffsetDynamics(8)()

	// Serialize static fields and offsets for dynamic ones (lazy fill)
	ssz.EncodeStaticBinaries(enc, h.BlockRoots[:])
	ssz.EncodeStaticBinaries(enc, h.StateRoots[:])
}

func (h *HistoricalBatch) DecodeSSZ(dec *ssz.Decoder) {
	// Initialize a dynamic decoder with the given starting offsets
	defer dec.OffsetDynamics(8)()

	// Serialize static fields and offsets for dynamic ones (lazy fill)
	ssz.DecodeStaticBinaries(dec, h.BlockRoots[:])
	ssz.DecodeStaticBinaries(dec, h.StateRoots[:])
}
