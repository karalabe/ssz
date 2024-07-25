// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package tests

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"testing"

	"github.com/karalabe/ssz"
	types "github.com/karalabe/ssz/tests/testtypes/consensus-spec-tests"
)

// Tests that decoding less or more data than requested will result in a failure.
func TestDecodeMissized(t *testing.T) {
	obj := new(testMissizedType)

	blob := make([]byte, ssz.Size(obj, ssz.ForkUnknown)+1)
	if err := ssz.DecodeFromBytes(blob, obj, ssz.ForkUnknown); !errors.Is(err, ssz.ErrObjectSlotSizeMismatch) {
		t.Errorf("decode from bytes error mismatch: have %v, want %v", err, ssz.ErrObjectSlotSizeMismatch)
	}
	if err := ssz.DecodeFromStream(bytes.NewReader(blob), obj, uint32(len(blob)), ssz.ForkUnknown); !errors.Is(err, ssz.ErrObjectSlotSizeMismatch) {
		t.Errorf("decode from stream error mismatch: have %v, want %v", err, ssz.ErrObjectSlotSizeMismatch)
	}

	blob = make([]byte, ssz.Size(obj, ssz.ForkUnknown)-1)
	if err := ssz.DecodeFromBytes(blob, obj, ssz.ForkUnknown); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("decode from bytes error mismatch: have %v, want %v", err, io.ErrUnexpectedEOF)
	}
	if err := ssz.DecodeFromStream(bytes.NewReader(blob), obj, uint32(len(blob)), ssz.ForkUnknown); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("decode from stream error mismatch: have %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

type testMissizedType struct {
	A, B uint64
}

func (t *testMissizedType) SizeSSZ(sizer *ssz.Sizer) uint32 { return 16 }
func (t *testMissizedType) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &t.A)
	ssz.DefineUint64(codec, &t.B)
}

// Tests that encoding more data than available space will result in a failure.
func TestEncodeOversized(t *testing.T) {
	obj := new(testMissizedType)

	blob := make([]byte, ssz.Size(obj, ssz.ForkUnknown)-1)
	if err := ssz.EncodeToBytes(blob, obj, ssz.ForkUnknown); !errors.Is(err, ssz.ErrBufferTooSmall) {
		t.Errorf("encode to bytes error mismatch: have %v, want %v", err, ssz.ErrBufferTooSmall)
	}
	if err := ssz.EncodeToStream(&testEncodeOversizedStream{blob}, obj, ssz.ForkUnknown); err == nil {
		t.Errorf("encode to stream error mismatch: have nil, want stream full") // wonky, but should be fine
	}
}

type testEncodeOversizedStream struct {
	sink []byte
}

func (s *testEncodeOversizedStream) Write(p []byte) (n int, err error) {
	// Keep writing until space runs out, then reject it
	copy(s.sink, p)

	n = len(p)
	if len(s.sink) < len(p) {
		n = len(s.sink)
	}
	s.sink = s.sink[n:]
	if n < len(p) {
		err = errors.New("stream full")
	}
	return n, err
}

// Tests that decoding an empty dynamic list via a non-empty container with an
// empty counter offset is rejected.
func TestZeroCounterOffset(t *testing.T) {
	inSSZ, err := hex.DecodeString("30303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030fc01000030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030fe010000303000000000")
	if err != nil {
		panic(err)
	}
	err = ssz.DecodeFromBytes(inSSZ, new(types.ExecutionPayload), ssz.ForkUnknown)
	if !errors.Is(err, ssz.ErrZeroCounterOffset) {
		t.Errorf("decode error mismatch: have %v, want %v", err, ssz.ErrZeroCounterOffset)
	}
}

// Tests that decoding a boolean with an invalid encoding is rejected.
func TestInvalidBoolean(t *testing.T) {
	inSSZ, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		panic(err)
	}
	err = ssz.DecodeFromBytes(inSSZ, new(types.Validator), ssz.ForkUnknown)
	if !errors.Is(err, ssz.ErrInvalidBoolean) {
		t.Errorf("decode error mismatch: have %v, want %v", err, ssz.ErrInvalidBoolean)
	}
}
