// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import (
	"github.com/karalabe/ssz"
)

type HistoricalBatchVariation struct {
	BlockRoots [8192]Hash
	StateRoots []Hash // Could be [8192]Hash, we're just testing the checked API like this
}

func (h *HistoricalBatchVariation) SizeSSZ() uint32 { return 2 * 8192 * 32 }
func (h *HistoricalBatchVariation) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineArrayOfStaticBytes(codec, h.BlockRoots[:])
	ssz.DefineCheckedArrayOfStaticBytes(codec, &h.StateRoots, 8192)
}
