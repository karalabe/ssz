// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type AttestationData struct {
	Slot            Slot
	Index           uint64
	BeaconBlockHash Hash
	Source          *Checkpoint
	Target          *Checkpoint
}

func (a *AttestationData) SizeSSZ() uint32 { return 128 }
func (a *AttestationData) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &a.Slot)                 // Field (0) - Slot             -  8 bytes
	ssz.DefineUint64(codec, &a.Index)                // Field (1) - Index            -  8 bytes
	ssz.DefineStaticBytes(codec, &a.BeaconBlockHash) // Field (2) - BeaconBlockHash  - 32 bytes
	ssz.DefineStaticObject(codec, &a.Source)         // Field (3) - Source           - 40 bytes
	ssz.DefineStaticObject(codec, &a.Target)         // Field (4) - Source           - 40 bytes
}
