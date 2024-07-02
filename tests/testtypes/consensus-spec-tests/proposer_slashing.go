// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type ProposerSlashing struct {
	Header1 *SignedBeaconBlockHeader
	Header2 *SignedBeaconBlockHeader
}

func (s *ProposerSlashing) SizeSSZ() uint32 { return 416 }
func (s *ProposerSlashing) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticObject(codec, &s.Header1) // Field (0) - Header1 - 208 bytes
	ssz.DefineStaticObject(codec, &s.Header2) // Field (1) - Header2 - 208 bytes
}
