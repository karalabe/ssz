// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type Eth1Data struct {
	DepositRoot  Hash
	DepositCount uint64
	BlockHash    Hash
}

func (d *Eth1Data) SizeSSZ() uint32 { return 72 }
func (d *Eth1Data) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticBytes(codec, &d.DepositRoot) // Field (0) - DepositRoot  - 32 bytes
	ssz.DefineUint64(codec, &d.DepositCount)     // Field (1) - DepositCount -  8 bytes
	ssz.DefineStaticBytes(codec, &d.BlockHash)   // Field (0) - BlockHash    - 32 bytes
}
