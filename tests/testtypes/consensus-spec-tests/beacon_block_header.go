// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type BeaconBlockHeader struct {
	Slot          uint64
	ProposerIndex uint64
	ParentRoot    Hash
	StateRoot     Hash
	BodyRoot      Hash
}

func (b *BeaconBlockHeader) SizeSSZ() uint32 { return 112 }
func (b *BeaconBlockHeader) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &b.Slot)            // Field (0) - Slot          -  8 bytes
	ssz.DefineUint64(codec, &b.ProposerIndex)   // Field (1) - ProposerIndex -  8 bytes
	ssz.DefineStaticBytes(codec, &b.ParentRoot) // Field (2) - ParentRoot    - 32 bytes
	ssz.DefineStaticBytes(codec, &b.StateRoot)  // Field (3) - StateRoot    - 32 bytes
	ssz.DefineStaticBytes(codec, &b.BodyRoot)   // Field (4) - BodyRoot    - 32 bytes
}
