// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type Attestation struct {
	AggregationBits []byte
	Data            *AttestationData
	Signature       [96]byte
}

func (a *Attestation) SizeSSZ(fixed bool) uint32 {
	size := uint32(228)
	if !fixed {
		size += ssz.SizeDynamicBytes(a.AggregationBits)
	}
	return size
}
func (a *Attestation) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineDynamicBytesOffset(codec, &a.AggregationBits, 2048) // Offset (0) - AggregationBits -  4 bytes
	ssz.DefineStaticObject(codec, &a.Data)                        // Field  (1) - Data            - 128 bytes
	ssz.DefineStaticBytes(codec, a.Signature[:])                  // Field  (2) - Signature       -  96 bytes

	ssz.DefineDynamicBytesContent(codec, &a.AggregationBits, 2048) // Offset (0) - AggregationBits -  4 bytes
}
