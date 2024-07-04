package tests

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/karalabe/ssz"
	types "github.com/karalabe/ssz/tests/testtypes/consensus-spec-tests"
)

// Tests that decoding less data than requested will result in a failure.
func TestDecodeUndersized(t *testing.T) {
	sobj := new(testDecodeUndersizedType)
	blob := make([]byte, sobj.SizeSSZ()+1)
	if err := ssz.DecodeFromBytes(blob, sobj); !errors.Is(err, ssz.ErrObjectSlotSizeMismatch) {
		t.Errorf("decode from bytes error mismatch: have %v, want %v", err, ssz.ErrObjectSlotSizeMismatch)
	}
	if err := ssz.DecodeFromStream(bytes.NewReader(blob), sobj, uint32(len(blob))); !errors.Is(err, ssz.ErrObjectSlotSizeMismatch) {
		t.Errorf("decode from stream error mismatch: have %v, want %v", err, ssz.ErrObjectSlotSizeMismatch)
	}
}

type testDecodeUndersizedType struct {
	A, B uint64
}

func (t *testDecodeUndersizedType) SizeSSZ() uint32 { return 16 }
func (t *testDecodeUndersizedType) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &t.A)
	ssz.DefineUint64(codec, &t.B)
}

// Tests that decoding an empty dynamic list via a non-empty container with an
// empty counter offset is rejected.
func TestZeroCounterOffset(t *testing.T) {
	inSSZ, err := hex.DecodeString("30303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030fc01000030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030fe010000303000000000")
	if err != nil {
		panic(err)
	}
	err = ssz.DecodeFromStream(bytes.NewReader(inSSZ), new(types.ExecutionPayload), uint32(len(inSSZ)))
	if !errors.Is(err, ssz.ErrZeroCounterOffset) {
		t.Errorf("decode error mismatch: have %v, want %v", err, ssz.ErrZeroCounterOffset)
	}
}
