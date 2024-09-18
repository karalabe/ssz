// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package tests

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/karalabe/ssz"
)

// testZeroValue does a bunch of encoding/decoding/hashing variations on the zero
// value of input types to check that the SSZ implementation can correctly handle
// the different uninitialized fields.
func testZeroValue[T newableObject[U], U any](t *testing.T, fork ssz.Fork) {
	// Verify that streaming/buffering encoding of a zero value results in the
	// same binary (maybe incorrect, we just want to see that they're the same).
	str1 := new(bytes.Buffer)
	if err := ssz.EncodeToStreamOnFork(str1, T(new(U)), fork); err != nil {
		t.Fatalf("failed to stream-encode zero-value object: %v", err)
	}
	bin1 := make([]byte, ssz.SizeOnFork(T(new(U)), fork))
	if err := ssz.EncodeToBytesOnFork(bin1, T(new(U)), fork); err != nil {
		t.Fatalf("failed to buffer-encode zero-value object: %v", err)
	}
	if !bytes.Equal(str1.Bytes(), bin1) {
		t.Fatalf("zero-value encoding mismatch: stream %x, buffer %x", str1, bin1)
	}
	// Decode the previous encoding in both streaming/buffering mode and check
	// that the produced objects are the same.
	obj1 := T(new(U))
	if err := ssz.DecodeFromStreamOnFork(bytes.NewReader(bin1), T(new(U)), uint32(len(bin1)), fork); err != nil {
		t.Fatalf("failed to stream-decode zero-value object: %v", err)
	}
	obj2 := T(new(U))
	if err := ssz.DecodeFromBytesOnFork(bin1, T(new(U)), fork); err != nil {
		t.Fatalf("failed to buffer-decode zero-value object: %v", err)
	}
	if !reflect.DeepEqual(obj1, obj2) {
		t.Fatalf("zero-value decoding mismatch: stream %+v, buffer %+v", obj1, obj2)
	}
	// We can't compare the decoded zero-value to the true zero-values as pointer
	// nil-ness might be different. To verify that the decoding was successful, do
	// yet another round of encodings and check that to the original ones.
	str2 := new(bytes.Buffer)
	if err := ssz.EncodeToStreamOnFork(str2, obj1, fork); err != nil {
		t.Fatalf("failed to stream-encode decoded object: %v", err)
	}
	bin2 := make([]byte, ssz.SizeOnFork(obj1, fork))
	if err := ssz.EncodeToBytesOnFork(bin2, obj1, fork); err != nil {
		t.Fatalf("failed to buffer-encode decoded object: %v", err)
	}
	if !bytes.Equal(str2.Bytes(), bin2) {
		t.Fatalf("re-encoding mismatch: stream %x, buffer %x", str2, bin2)
	}
	if !bytes.Equal(bin1, bin2) {
		t.Fatalf("re-encoding mismatch: zero-value %x, decoded %x", bin1, bin2)
	}
	// Encoding/decoding seems to work, hash the zero-value and re-encoded value
	// in both sequential/concurrent more and verify the results.
	hashes := map[string][32]byte{
		"zero-value-sequential": ssz.HashSequentialOnFork(T(new(U)), fork),
		"zero-value-concurrent": ssz.HashConcurrentOnFork(T(new(U)), fork),
		"decoded-sequential":    ssz.HashSequentialOnFork(obj1, fork),
		"decoded-concurrent":    ssz.HashSequentialOnFork(obj1, fork),
	}
	for key1, hash1 := range hashes {
		for key2, hash2 := range hashes {
			if hash1 != hash2 {
				t.Errorf("hash mismatch: %s %x, %s %x", key1, hash1, key2, hash2)
			}
		}
	}
}
