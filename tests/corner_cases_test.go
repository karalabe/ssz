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

	blob := make([]byte, ssz.Size(obj)+1)
	if err := ssz.DecodeFromBytes(blob, obj); !errors.Is(err, ssz.ErrObjectSlotSizeMismatch) {
		t.Errorf("decode from bytes error mismatch: have %v, want %v", err, ssz.ErrObjectSlotSizeMismatch)
	}
	if err := ssz.DecodeFromStream(bytes.NewReader(blob), obj, uint32(len(blob))); !errors.Is(err, ssz.ErrObjectSlotSizeMismatch) {
		t.Errorf("decode from stream error mismatch: have %v, want %v", err, ssz.ErrObjectSlotSizeMismatch)
	}

	blob = make([]byte, ssz.Size(obj)-1)
	if err := ssz.DecodeFromBytes(blob, obj); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("decode from bytes error mismatch: have %v, want %v", err, io.ErrUnexpectedEOF)
	}
	if err := ssz.DecodeFromStream(bytes.NewReader(blob), obj, uint32(len(blob))); !errors.Is(err, io.ErrUnexpectedEOF) {
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

	blob := make([]byte, ssz.Size(obj)-1)
	if err := ssz.EncodeToBytes(blob, obj); !errors.Is(err, ssz.ErrBufferTooSmall) {
		t.Errorf("encode to bytes error mismatch: have %v, want %v", err, ssz.ErrBufferTooSmall)
	}
	if err := ssz.EncodeToStream(&testEncodeOversizedStream{blob}, obj); err == nil {
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
	err = ssz.DecodeFromBytes(inSSZ, new(types.ExecutionPayload))
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
	err = ssz.DecodeFromBytes(inSSZ, new(types.Validator))
	if !errors.Is(err, ssz.ErrInvalidBoolean) {
		t.Errorf("decode error mismatch: have %v, want %v", err, ssz.ErrInvalidBoolean)
	}
}

// Tests that decoding empty slices will init them instead of leaving as nil.
func TestEmptySliceInit(t *testing.T) {
	obj := new(testEmptySlicesType)
	buf := new(bytes.Buffer)

	if err := ssz.EncodeToStream(buf, obj); err != nil {
		panic(err)
	}
	if err := ssz.DecodeFromBytes(buf.Bytes(), obj); err != nil {
		panic(err)
	}
	if obj.A == nil {
		t.Errorf("failed to init empty uint64 slice")
	}
	if obj.B == nil {
		t.Errorf("failed to init empty statc bytes slice")
	}
	if obj.C == nil {
		t.Errorf("failed to init empty dynamic bytes slice")
	}
	if obj.D == nil {
		t.Errorf("failed to init empty static objects slice")
	}
	if obj.E == nil {
		t.Errorf("failed to init empty dynamic objects slice")
	}
}

type testEmptySlicesType struct {
	A []uint64                  // Slice of uint64
	B [][32]byte                // Slice of static bytes
	C [][]byte                  // Slice of dynamic bytes
	D []*types.Withdrawal       // Slice of static objects
	E []*types.ExecutionPayload // Slice of dynamic objects
}

func (t *testEmptySlicesType) SizeSSZ(sizer *ssz.Sizer, fixed bool) (size uint32) {
	size = 5 * 4
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfUint64s(sizer, t.A)
	size += ssz.SizeSliceOfStaticBytes(sizer, t.B)
	size += ssz.SizeSliceOfDynamicBytes(sizer, t.C)
	size += ssz.SizeSliceOfStaticObjects(sizer, t.D)
	size += ssz.SizeSliceOfDynamicObjects(sizer, t.E)

	return size
}
func (t *testEmptySlicesType) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineSliceOfUint64sOffset(codec, &t.A, 16)
	ssz.DefineSliceOfStaticBytesOffset(codec, &t.B, 16)
	ssz.DefineSliceOfDynamicBytesOffset(codec, &t.C, 16, 16)
	ssz.DefineSliceOfStaticObjectsOffset(codec, &t.D, 16)
	ssz.DefineSliceOfDynamicObjectsOffset(codec, &t.E, 16)

	ssz.DefineSliceOfUint64sContent(codec, &t.A, 16)
	ssz.DefineSliceOfStaticBytesContent(codec, &t.B, 16)
	ssz.DefineSliceOfDynamicBytesContent(codec, &t.C, 16, 16)
	ssz.DefineSliceOfStaticObjectsContent(codec, &t.D, 16)
	ssz.DefineSliceOfDynamicObjectsContent(codec, &t.E, 16)
}
