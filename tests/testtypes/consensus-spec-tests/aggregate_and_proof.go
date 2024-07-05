// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type AggregateAndProof struct {
	Index          uint64
	Aggregate      *Attestation
	SelectionProof [96]byte
}

func (a *AggregateAndProof) SizeSSZ(fixed bool) uint32 {
	size := uint32(108)
	if !fixed {
		size += ssz.SizeDynamicObject(a.Aggregate)
	}
	return size
}
func (a *AggregateAndProof) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &a.Index)
	ssz.DefineDynamicObjectOffset(codec, &a.Aggregate)
	ssz.DefineStaticBytes(codec, a.SelectionProof[:])

	ssz.DefineDynamicObjectContent(codec, &a.Aggregate)
}
