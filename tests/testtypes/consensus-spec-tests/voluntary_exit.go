// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type VoluntaryExit struct {
	Epoch          uint64
	ValidatorIndex uint64
}

func (v *VoluntaryExit) SizeSSZ() uint32 { return 16 }
func (v *VoluntaryExit) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &v.Epoch)          // Field (0) - Epoch          - 8 bytes
	ssz.DefineUint64(codec, &v.ValidatorIndex) // Field (1) - ValidatorIndex - 8 bytes
}
