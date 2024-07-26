// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/prysmaticlabs/go-bitfield"

//go:generate go run -cover ../../../cmd/sszgen -type SingleFieldTestStruct -out gen_single_field_test_struct_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type SmallTestStruct -out gen_small_test_struct_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type FixedTestStruct -out gen_fixed_test_struct_ssz.go
//go:generate go run -cover ../../../cmd/sszgen -type BitsStruct -out gen_bits_struct_ssz.go

type SingleFieldTestStruct struct {
	A byte
}

type SmallTestStruct struct {
	A uint16
	B uint16
}

type FixedTestStruct struct {
	A uint8
	B uint64
	C uint32
}

type BitsStruct struct {
	A bitfield.Bitlist `ssz-max:"5"`
	B [1]byte          `ssz-size:"2" ssz:"bits"`
	C [1]byte          `ssz-size:"1" ssz:"bits"`
	D bitfield.Bitlist `ssz-max:"6"`
	E [1]byte          `ssz-size:"8" ssz:"bits"`
}
