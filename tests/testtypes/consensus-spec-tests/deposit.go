// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type Deposit struct {
	Proof [33][32]byte
	Data  *DepositData
}

func (d *Deposit) SizeSSZ() uint32 { return 33*32 + 184 }
func (d *Deposit) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineArrayOfStaticBytes(codec, d.Proof[:]) // Field (0) - Proof - 1056 bytes
	ssz.DefineStaticObject(codec, &d.Data)          // Field (1) - Data  -  184 bytes
}
