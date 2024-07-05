// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type SignedVoluntaryExit struct {
	Exit      *VoluntaryExit `json:"message"`
	Signature [96]byte       `json:"signature" ssz-size:"96"`
}

func (v *SignedVoluntaryExit) SizeSSZ() uint32 { return 112 }
func (v *SignedVoluntaryExit) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticObject(codec, &v.Exit)       // Field (0) - Exit          - 16 bytes
	ssz.DefineStaticBytes(codec, v.Signature[:]) // Field (1) - Signature - 96 bytes
}
