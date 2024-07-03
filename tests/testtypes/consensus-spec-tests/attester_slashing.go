// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type AttesterSlashing struct {
	Attestation1 *IndexedAttestation `json:"attestation_1"`
	Attestation2 *IndexedAttestation `json:"attestation_2"`
}

func (a *AttesterSlashing) SizeSSZ(fixed bool) uint32 {
	size := uint32(8)
	if !fixed {
		size += ssz.SizeDynamicObject(a.Attestation1)
		size += ssz.SizeDynamicObject(a.Attestation2)
	}
	return size
}
func (a *AttesterSlashing) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineDynamicObjectOffset(codec, &a.Attestation1) // Offset (0) - Attestation1 - 4 bytes
	ssz.DefineDynamicObjectOffset(codec, &a.Attestation2) // Offset (1) - Attestation2 - 4 bytes

	ssz.DefineDynamicObjectContent(codec, &a.Attestation1) // Offset (0) - Attestation1 - 4 bytes
	ssz.DefineDynamicObjectContent(codec, &a.Attestation2) // Offset (1) - Attestation2 - 4 bytes
}
