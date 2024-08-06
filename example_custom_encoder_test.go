// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz_test

import (
	"fmt"
	"sync"

	"github.com/karalabe/ssz"
)

type WithdrawalCustomCodec struct {
	Index     uint64 `ssz-size:"8"`
	Validator uint64 `ssz-size:"8"`
	Address   []byte `ssz-size:"20"`
	Amount    uint64 `ssz-size:"8"`
}

func (w *WithdrawalCustomCodec) SizeSSZ() uint32 { return 44 }

func (w *WithdrawalCustomCodec) DefineSSZ(codec *CustomCodec) {
	ssz.DefineUint64(codec, &w.Index)                   // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)               // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineCheckedStaticBytes(codec, &w.Address, 20) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)                  // Field (3) - Amount         -  8 bytes
}

func ExampleCustomEncoder() {
	ssz.UpdateGlobalHasherPool(&sync.Pool{
		New: func() any {
			codec := &CustomCodec{}
			codec.dec = (&ssz.Decoder[*CustomCodec]{}).WithCodec(codec)
			return codec
		},
	})
	hash := ssz.HashSequential(new(WithdrawalCustomCodec))

	fmt.Printf("hash: %#x\n", hash)
	// Output
	// hash: 0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71
}

/* -------------------------------------------------------------------------- */
/*                              Custom Codec Impl                             */
/* -------------------------------------------------------------------------- */

type CustomCodec struct {
	enc *ssz.Encoder[*CustomCodec]
	dec *ssz.Decoder[*CustomCodec]
	has *ssz.Hasher[*CustomCodec]
}

// Enc returns the Encoder associated with the CustomCodec.
func (c *CustomCodec) Enc() *ssz.Encoder[*CustomCodec] {
	return c.enc
}

// SetEncoder sets the Encoder for the CustomCodec.
func (c *CustomCodec) SetEncoder(enc *ssz.Encoder[*CustomCodec]) {
	c.enc = enc
}

// Dec returns the Decoder associated with the CustomCodec.
func (c *CustomCodec) Dec() *ssz.Decoder[*CustomCodec] {
	return c.dec
}

// SetDecoder sets the Decoder for the CustomCodec.
func (c *CustomCodec) SetDecoder(dec *ssz.Decoder[*CustomCodec]) {
	c.dec = dec
}

// Has returns the Hasher associated with the CustomCodec.
func (c *CustomCodec) Has() *ssz.Hasher[*CustomCodec] {
	return c.has
}

// SetHasher sets the Hasher for the CustomCodec.
func (c *CustomCodec) SetHasher(has *ssz.Hasher[*CustomCodec]) {
	c.has = has
}

// DefineEncoder uses a dedicated encoder in case the types SSZ conversion is for
// some reason asymmetric (e.g. encoding depends on fields, decoding depends on
// outer context).
//
// In reality, it will be the live code run when the object is being serialized.
func (c *CustomCodec) DefineEncoder(impl func(enc *ssz.Encoder[*CustomCodec])) {
	if c.enc != nil {
		impl(c.enc)
	}
}

// DefineDecoder uses a dedicated decoder in case the types SSZ conversion is for
// some reason asymmetric (e.g. encoding depends on fields, decoding depends on
// outer context).
//
// In reality, it will be the live code run when the object is being parsed.
func (c *CustomCodec) DefineDecoder(impl func(dec *ssz.Decoder[*CustomCodec])) {
	if c.dec != nil {
		impl(c.dec)
	}
}

// DefineHasher uses a dedicated hasher in case the types SSZ conversion is for
// some reason asymmetric (e.g. encoding depends on fields, decoding depends on
// outer context).
//
// In reality, it will be the live code run when the object is being parsed.
func (c *CustomCodec) DefineHasher(impl func(has *ssz.Hasher[*CustomCodec])) {
	if c.has != nil {
		impl(c.has)
	}
}
