// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type SignedBeaconBlockHeader struct {
	Header    *BeaconBlockHeader
	Signature [96]byte
}

func (s *SignedBeaconBlockHeader) SizeSSZ() uint32 { return 208 }
func (s *SignedBeaconBlockHeader) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticObject(codec, &s.Header)   // Field (0) - Header    - 112 bytes
	ssz.DefineStaticBytes(codec, &s.Signature) // Field (1) - Signature -  96 bytes
}
