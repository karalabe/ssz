// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

func (w *Withdrawal) SizeSSZ() uint32 { return 44 }
func (w *Withdrawal) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &w.Index)          // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)      // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineStaticBytes(codec, w.Address[:]) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)         // Field (3) - Amount         -  8 bytes
}
