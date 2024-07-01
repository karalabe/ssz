// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

// Hash is a standalone mock of go-ethereum;s common.Hash
type Hash [32]byte

// Address is a standalone mock of go-ethereum's common.Address
type Address [20]byte

// LogsBloom is a standalone mock of go-ethereum's types.LogsBloom
type LogsBloom [256]byte
