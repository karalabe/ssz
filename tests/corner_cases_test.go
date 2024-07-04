package tests

import (
	"bytes"
	"errors"
	"testing"

	"github.com/karalabe/ssz"
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
