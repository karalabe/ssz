// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type WithdrawalVariation struct {
	Index     uint64 `ssz-size:"8"`
	Validator uint64 `ssz-size:"8"`
	Address   []byte `ssz-size:"20"`
	Amount    uint64 `ssz-size:"8"`
}

func (w *WithdrawalVariation) SizeSSZ() uint32 { return 44 }
func (w *WithdrawalVariation) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &w.Index)                   // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)               // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineCheckedStaticBytes(codec, &w.Address, 20) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)                  // Field (3) - Amount         -  8 bytes
}
