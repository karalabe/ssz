// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type WithdrawalDeneb struct {
	Index     uint64
	Validator uint64
	Address   Address
	Amount    uint64
}

func (w *WithdrawalDeneb) StaticSSZ() bool { return true }
func (w *WithdrawalDeneb) SizeSSZ() uint32 { return 44 }

func (w *WithdrawalDeneb) EncodeSSZ(enc *ssz.Encoder) {
	ssz.EncodeUint64(enc, w.Index)      // Field (0) - Index          -  8 bytes
	ssz.EncodeUint64(enc, w.Validator)  // Field (1) - ValidatorIndex -  8 bytes
	ssz.EncodeBinary(enc, w.Address[:]) // Field (2) - Address        - 20 bytes
	ssz.EncodeUint64(enc, w.Amount)     // Field (3) - Amount         -  8 bytes
}

func (w *WithdrawalDeneb) DecodeSSZ(dec *ssz.Decoder) {
	ssz.DecodeUint64(dec, &w.Index)     // Field (0) - Index          -  8 bytes
	ssz.DecodeUint64(dec, &w.Validator) // Field (1) - ValidatorIndex -  8 bytes
	ssz.DecodeBinary(dec, w.Address[:]) // Field (2) - Address        - 20 bytes
	ssz.DecodeUint64(dec, &w.Amount)    // Field (3) - Amount         -  8 bytes
}
