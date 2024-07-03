// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type BeaconBlockBody struct {
	RandaoReveal      [96]byte               `json:"randao_reveal" ssz-size:"96"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          [32]byte               `json:"graffiti" ssz-size:"32"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings" ssz-max:"16"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings" ssz-max:"2"`
	Attestations      []*Attestation         `json:"attestations" ssz-max:"128"`
	Deposits          []*Deposit             `json:"deposits" ssz-max:"16"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits" ssz-max:"16"`
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
