// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

//go:generate go run ../../../cmd/sszgen -type WithdrawalVariation -out gen_withdrawal_variation_ssz.go

type WithdrawalVariation struct {
	Index     uint64
	Validator uint64
	Address   []byte `ssz-size:"20"` // Static type defined via ssz-size tag
	Amount    uint64
}
