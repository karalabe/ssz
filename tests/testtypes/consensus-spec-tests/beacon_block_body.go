// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type BeaconBlockBody struct {
	RandaoReveal      [96]byte
	Eth1Data          *Eth1Data
	Graffiti          [32]byte
	ProposerSlashings []*ProposerSlashing
	AttesterSlashings []*AttesterSlashing
	Attestations      []*Attestation
	Deposits          []*Deposit
	VoluntaryExits    []*SignedVoluntaryExit
}

func (b *BeaconBlockBody) SizeSSZ(fixed bool) uint32 {
	size := uint32(220)
	if !fixed {
		size += ssz.SizeSliceOfStaticObjects(b.ProposerSlashings)
		size += ssz.SizeSliceOfDynamicObjects(b.AttesterSlashings)
		size += ssz.SizeSliceOfDynamicObjects(b.Attestations)
		size += ssz.SizeSliceOfStaticObjects(b.Deposits)
		size += ssz.SizeSliceOfStaticObjects(b.VoluntaryExits)
	}
	return size
}
func (b *BeaconBlockBody) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticBytes(codec, b.RandaoReveal[:])
	ssz.DefineStaticObject(codec, &b.Eth1Data)
	ssz.DefineStaticBytes(codec, b.Graffiti[:])
	ssz.DefineSliceOfStaticObjects(codec, &b.ProposerSlashings, 16)
	ssz.DefineSliceOfDynamicObjects(codec, &b.AttesterSlashings, 2)
	ssz.DefineSliceOfDynamicObjects(codec, &b.Attestations, 128)
	ssz.DefineSliceOfStaticObjects(codec, &b.Deposits, 16)
	ssz.DefineSliceOfStaticObjects(codec, &b.VoluntaryExits, 16)
}
