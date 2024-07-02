// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type IndexedAttestation struct {
	AttestationIndices []uint64
	Data               *AttestationData
	Signature          [96]byte
}

func (a *IndexedAttestation) SizeSSZ() uint32 {
	size := uint32(228)
	size += ssz.SizeSliceOfUint64s(a.AttestationIndices)
	return size
}
func (a *IndexedAttestation) DefineSSZ(codec *ssz.Codec) {
	defer codec.OffsetDynamics(228)()

	ssz.DefineSliceOfUint64s(codec, &a.AttestationIndices, 2048) // Offset (0) - AttestationIndices - 4 bytes
	ssz.DefineStaticObject(codec, &a.Data)                       // Field (1) - Data      - 128 bytes
	ssz.DefineStaticBytes(codec, a.Signature[:])                 // Field (2) - Signature - 96 bytes
}
