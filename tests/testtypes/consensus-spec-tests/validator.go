// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type Validator struct {
	Pubkey                     [48]byte
	WithdrawalCredentials      [32]byte
	EffectiveBalance           uint64
	Slashed                    bool
	ActivationEligibilityEpoch uint64
	ActivationEpoch            uint64
	ExitEpoch                  uint64
	WithdrawableEpoch          uint64
}

func (v *Validator) SizeSSZ() uint32 { return 121 }
func (v *Validator) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticBytes(codec, &v.Pubkey)
	ssz.DefineStaticBytes(codec, &v.WithdrawalCredentials)
	ssz.DefineUint64(codec, &v.EffectiveBalance)
	ssz.DefineBool(codec, &v.Slashed)
	ssz.DefineUint64(codec, &v.ActivationEligibilityEpoch)
	ssz.DefineUint64(codec, &v.ActivationEpoch)
	ssz.DefineUint64(codec, &v.ExitEpoch)
	ssz.DefineUint64(codec, &v.WithdrawableEpoch)
}
