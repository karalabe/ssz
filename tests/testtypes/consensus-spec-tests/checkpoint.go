// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type Checkpoint struct {
	Epoch uint64
	Root  Hash
}

func (c *Checkpoint) SizeSSZ() uint32 { return 40 }
func (c *Checkpoint) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &c.Epoch)       // Field (0) - Epoch -  8 bytes
	ssz.DefineStaticBytes(codec, c.Root[:]) // Field (1) - Root  - 32 bytes
}
