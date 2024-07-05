// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type DepositData struct {
	Pubkey                [48]byte
	WithdrawalCredentials [32]byte
	Amount                uint64
	Signature             [96]byte
	Root                  [32]byte
}

func (d *DepositData) SizeSSZ() uint32 { return 184 }
func (d *DepositData) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticBytes(codec, &d.Pubkey)                // Field (0) - Pubkey                - 48 bytes
	ssz.DefineStaticBytes(codec, &d.WithdrawalCredentials) // Field (1) - WithdrawalCredentials - 32 bytes
	ssz.DefineUint64(codec, &d.Amount)                     // Field (2) - Amount                - 32 bytes
	ssz.DefineStaticBytes(codec, &d.Signature)             // Field (3) - Signature             - 32 bytes
}
