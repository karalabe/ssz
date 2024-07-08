// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"go/types"
)

// opset is a group of methods that define how different pieces of an ssz codec
// operates on a given type. It may be static or dynamic.
type opset interface {
	defineStatic() string  // get the op for the static section
	defineDynamic() string // get the op for the dynamic section (optional)
	encodeStatic() string  // get the op for the static section
	encodeDynamic() string // get the op for the dynamic section (optional)
	decodeStatic() string  // get the op for the static section
	decodeDynamic() string // get the op for the dynamic section (optional)
}

// opsetStatic is a group of methods that define how different pieces of an ssz
// codec operates on a given static type. Ideally these would be some go/types
// function values, but alas too much pain, especially with generics.
type opsetStatic struct {
	define string // DefineXYZ method for the ssz.Codec
	encode string // EncodeXYZ method for the ssz.Encoder
	decode string // DecodeXYZ method for the ssz.Decoder
	bytes  int    // Number of bytes in the ssz encoding (0 == unknown)
}

func (os *opsetStatic) defineStatic() string  { return os.define }
func (os *opsetStatic) encodeStatic() string  { return os.encode }
func (os *opsetStatic) decodeStatic() string  { return os.decode }
func (os *opsetStatic) defineDynamic() string { return "" }
func (os *opsetStatic) encodeDynamic() string { return "" }
func (os *opsetStatic) decodeDynamic() string { return "" }

// opsetDynamic is a group of methods that define how different pieces of an ssz
// codec operates on a given dynamic type. Ideally these would be some go/types
// function values, but alas too much pain, especially with generics.
type opsetDynamic struct {
	defineOffset  string // DefineXYZOffset method for the ssz.Codec
	defineContent string // DefineXYZContent method for the ssz.Codec
	encodeOffset  string // EncodeXYZOffset method for the ssz.Encoder
	encodeContent string // EncodeXYZContent method for the ssz.Encoder
	decodeOffset  string // DecodeXYZOffset method for the ssz.Decoder
	decodeContent string // DecodeXYZContent method for the ssz.Decoder
}

func (os *opsetDynamic) defineStatic() string  { return os.defineOffset }
func (os *opsetDynamic) encodeStatic() string  { return os.encodeOffset }
func (os *opsetDynamic) decodeStatic() string  { return os.decodeOffset }
func (os *opsetDynamic) defineDynamic() string { return os.defineContent }
func (os *opsetDynamic) encodeDynamic() string { return os.encodeContent }
func (os *opsetDynamic) decodeDynamic() string { return os.decodeContent }

// resolveBasicOpset retrieves the opset required to handle a basic struct
// field. Yes, we could maybe have some of these be "computed" instead of hard
// coded, but it makes things brittle for corner-cases.
func (p *parseContext) resolveBasicOpset(typ *types.Basic) (*opsetStatic, error) {
	switch typ.Kind() {
	case types.Bool:
		return &opsetStatic{"DefineBool", "EncodeBool", "DecodeBool", 1}, nil
	case types.Uint64:
		return &opsetStatic{"DefineUint64", "EncodeUint64", "DecodeUint64", 8}, nil
	default:
		return nil, fmt.Errorf("unsupported basic type: %s", typ)
	}
}

// resolveUint256Opset retrieves the opset required to handle a uint256
// struct field implemented using github.com/holiman/uint256. Yay hard code!
func (p *parseContext) resolveUint256Opset() *opsetStatic {
	return &opsetStatic{"DefineUint256", "EncodeUint256", "DecodeUint256", 32}
}

func (p *parseContext) resolveArrayOpset(typ types.Type, size int) (*opsetStatic, error) {
	switch typ := typ.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Byte:
			return &opsetStatic{"DefineStaticBytes", "EncodeStaticBytes", "DecodeStaticBytes", size}, nil
		default:
			return nil, fmt.Errorf("unsupported array item basic type: %s", typ)
		}
	default:
		return nil, fmt.Errorf("unsupported array item type: %s", typ)
	}
}

func (p *parseContext) resolveSliceOpset(typ types.Type) (*opsetDynamic, error) {
	switch typ := typ.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Byte:
			return &opsetDynamic{
				"DefineDynamicBytesOffset", "DefineDynamicBytesContent",
				"EncodeDynamicBytesOffset", "EncodeDynamicBytesContent",
				"DecodeDynamicBytesOffset", "DecodeDynamicBytesContent"}, nil
		default:
			return nil, fmt.Errorf("unsupported slice item basic type: %s", typ)
		}
	default:
		return nil, fmt.Errorf("unsupported slice item type: %s", typ)
	}
}

func (p *parseContext) resolvePointerOpset(typ *types.Pointer) (opset, error) {
	if isUint256(typ.Elem()) {
		return &opsetStatic{"DefineUint256", "EncodeUint256", "DecodeUint256", 32}, nil
	}
	if types.Implements(typ, p.staticObjectIface) {
		return &opsetStatic{"DefineStaticObject", "EncodeStaticObject", "DecodeStaticObject", 0}, nil
	}
	if types.Implements(typ, p.dynamicObjectIface) {
		return &opsetDynamic{
			"DefineDynamicObjectOffset", "DefineDynamicObjectContent",
			"EncodeDynamicObjectOffset", "EncodeDynamicObjectContent",
			"DecodeDynamicObjectOffset", "DecodeDynamicObjectContent"}, nil
	}
	return nil, fmt.Errorf("unsupported pointer type %s", typ.String())
}
