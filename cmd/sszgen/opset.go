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
type opset interface{}

// opsetStatic is a group of methods that define how different pieces of an ssz
// codec operates on a given static type. Ideally these would be some go/types
// function values, but alas too much pain, especially with generics.
type opsetStatic struct {
	define string // DefineXYZ method for the ssz.Codec
	encode string // EncodeXYZ method for the ssz.Encoder
	decode string // DecodeXYZ method for the ssz.Decoder
	bytes  []int  // Number of bytes in the ssz encoding (0 == unknown)
}

// opsetDynamic is a group of methods that define how different pieces of an ssz
// codec operates on a given dynamic type. Ideally these would be some go/types
// function values, but alas too much pain, especially with generics.
type opsetDynamic struct {
	size          string // SizeXYZ method for the SizeSSZ method
	defineOffset  string // DefineXYZOffset method for the ssz.Codec
	defineContent string // DefineXYZContent method for the ssz.Codec
	encodeOffset  string // EncodeXYZOffset method for the ssz.Encoder
	encodeContent string // EncodeXYZContent method for the ssz.Encoder
	decodeOffset  string // DecodeXYZOffset method for the ssz.Decoder
	decodeContent string // DecodeXYZContent method for the ssz.Decoder
	sizes         []int  // Static item sizes for different dimensions
	limits        []int  // Maximum dynamic item sizes for different dimensions
}

// resolveBasicOpset retrieves the opset required to handle a basic struct
// field. Yes, we could maybe have some of these be "computed" instead of hard
// coded, but it makes things brittle for corner-cases.
func (p *parseContext) resolveBasicOpset(typ *types.Basic, tags *sizeTag) (opset, error) {
	// Sanity check a few tag constraints relevant for all basic types
	if tags != nil {
		if tags.limit != nil {
			return nil, fmt.Errorf("basic type cannot have ssz-max tag")
		}
		if len(tags.size) != 1 {
			return nil, fmt.Errorf("basic type requires 1D ssz-size tag: have %v", tags.size)
		}
	}
	// Return the type-specific opsets
	switch typ.Kind() {
	case types.Bool:
		if tags != nil && tags.size[0] != 1 {
			return nil, fmt.Errorf("boolean basic type requires ssz-size=1: have %d", tags.size[0])
		}
		return &opsetStatic{
			"DefineBool({{.Codec}}, &{{.Field}})",
			"EncodeBool({{.Codec}}, &{{.Field}})",
			"DecodeBool({{.Codec}}, &{{.Field}})",
			[]int{1},
		}, nil
	case types.Uint64:
		if tags != nil && tags.size[0] != 8 {
			return nil, fmt.Errorf("uint64 basic type requires ssz-size=8: have %d", tags.size[0])
		}
		return &opsetStatic{
			"DefineUint64({{.Codec}}, &{{.Field}})",
			"EncodeUint64({{.Codec}}, &{{.Field}})",
			"DecodeUint64({{.Codec}}, &{{.Field}})",
			[]int{8},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported basic type: %s", typ)
	}
}

func (p *parseContext) resolveBitlistOpset(tags *sizeTag) (opset, error) {
	if tags == nil || tags.limit == nil {
		return nil, fmt.Errorf("slice of bits type requires ssz-max tag")
	}
	if len(tags.size) > 0 {
		return nil, fmt.Errorf("slice of bits type cannot have ssz-size tag")
	}
	if len(tags.limit) != 1 {
		return nil, fmt.Errorf("slice of bits tag conflict: field supports [N] bits, tag wants %v bits", tags.limit)
	}
	return &opsetDynamic{
		"SizeSliceOfBits({{.Field}})",
		"DefineSliceOfBitsOffset({{.Codec}}, &{{.Field}})",
		fmt.Sprintf("DefineSliceOfBitsContent({{.Codec}}, &{{.Field}}, %d)", tags.limit[0]), // inject bit-cap directly
		"EncodeSliceOfBitsOffset({{.Codec}}, &{{.Field}})",
		fmt.Sprintf("EncodeSliceOfBitsContent({{.Codec}}, &{{.Field}}, %d)", tags.limit[0]), // inject bit-cap directly
		"DecodeSliceOfBitsOffset({{.Codec}}, &{{.Field}})",
		fmt.Sprintf("DecodeSliceOfBitsContent({{.Codec}}, &{{.Field}}, %d)", tags.limit[0]), // inject bit-cap directly
		nil, []int{(tags.limit[0] + 7) / 8},
	}, nil
}

func (p *parseContext) resolveArrayOpset(typ types.Type, name string, size int, tags *sizeTag) (opset, error) {
	switch typ := typ.(type) {
	case *types.Basic:
		// Sanity check a few tag constraints relevant for all arrays of basic types
		if tags != nil {
			if tags.limit != nil {
				return nil, fmt.Errorf("array of basic type cannot have ssz-max tag")
			}
		}
		switch typ.Kind() {
		case types.Byte:
			// If the byte array is a packet bitvector, handle is explicitly
			if tags != nil && tags.bits {
				if len(tags.size) != 1 || tags.size[0] < (size-1)*8+1 || tags.size[0] > size*8 {
					return nil, fmt.Errorf("array of bits tag conflict: field supports %d-%d bits, tag wants %v bits", (size-1)*8+1, size*8, tags.size)
				}
				return &opsetStatic{
					fmt.Sprintf("DefineArrayOfBits({{.Codec}}, &{{.Field}}, %d)", tags.size[0]), // inject bit-size directly
					fmt.Sprintf("EncodeArrayOfBits({{.Codec}}, &{{.Field}}, %d)", tags.size[0]), // inject bit-size directly
					fmt.Sprintf("DecodeArrayOfBits({{.Codec}}, &{{.Field}}, %d)", tags.size[0]), // inject bit-size directly
					[]int{size},
				}, nil
			}
			// Not a bitvector, interpret as plain byte array
			if tags != nil {
				if (len(tags.size) != 1 && len(tags.size) != 2) ||
					(len(tags.size) == 1 && tags.size[0] != size) ||
					(len(tags.size) == 2 && (tags.size[0] != size || tags.size[1] != 1)) {
					return nil, fmt.Errorf("array of byte basic type tag conflict: field is %d bytes, tag wants %v bytes", size, tags.size)
				}
			}
			return &opsetStatic{
				"DefineStaticBytes({{.Codec}}, &{{.Field}})",
				"EncodeStaticBytes({{.Codec}}, &{{.Field}})",
				"DecodeStaticBytes({{.Codec}}, &{{.Field}})",
				[]int{size},
			}, nil

		case types.Uint64:
			if tags != nil {
				if (len(tags.size) != 1 && len(tags.size) != 2) ||
					(len(tags.size) == 1 && tags.size[0] != size) ||
					(len(tags.size) == 2 && (tags.size[0] != size || tags.size[1] != 8)) {
					return nil, fmt.Errorf("array of byte basic type tag conflict: field is %d bytes, tag wants %v bytes", size, tags.size)
				}
			}
			return &opsetStatic{
				"DefineArrayOfUint64s({{.Codec}}, &{{.Field}})",
				"EncodeArrayOfUint64s({{.Codec}}, &{{.Field}})",
				"DecodeArrayOfUint64s({{.Codec}}, &{{.Field}})",
				[]int{size, 8},
			}, nil

		default:
			return nil, fmt.Errorf("unsupported array item basic type: %s", typ)
		}
	case *types.Array:
		return p.resolveArrayOfArrayOpset(typ.Elem(), name, size, typ.String(), int(typ.Len()), tags)

	case *types.Named:
		// For named arrays, we need to pass the name too for the generics
		if t, ok := typ.Underlying().(*types.Array); ok {
			return p.resolveArrayOfArrayOpset(t.Elem(), name, size, typ.Obj().Name(), int(t.Len()), tags)
		}
		return p.resolveArrayOpset(typ.Underlying(), name, size, tags)

	default:
		return nil, fmt.Errorf("unsupported array item type: %s", typ)
	}
}

func (p *parseContext) resolveArrayOfArrayOpset(typ types.Type, outerType string, outerSize int, innerType string, innerSize int, tags *sizeTag) (opset, error) {
	switch typ := typ.(type) {
	case *types.Basic:
		// Sanity check a few tag constraints relevant for all arrays of basic types
		if tags != nil {
			if tags.limit != nil {
				return nil, fmt.Errorf("array of array of basic type cannot have ssz-max tag")
			}
		}
		switch typ.Kind() {
		case types.Byte:
			if tags != nil {
				if (len(tags.size) != 2 && len(tags.size) != 3) ||
					(len(tags.size) == 2 && (tags.size[0] != outerSize || tags.size[1] != innerSize)) ||
					(len(tags.size) == 3 && (tags.size[0] != outerSize || tags.size[1] != innerSize || tags.size[2] != 1)) {
					return nil, fmt.Errorf("array of array of byte basic type tag conflict: field is [%d, %d] bytes, tag wants %v bytes", outerSize, innerSize, tags.size)
				}
			}
			if outerType == "" {
				outerType = fmt.Sprintf("[%d]%s", outerSize, innerType)
			}
			return &opsetStatic{
				fmt.Sprintf("DefineArrayOfStaticBytes[%s, %s]({{.Codec}}, &{{.Field}})", outerType, innerType),
				fmt.Sprintf("EncodeArrayOfStaticBytes[%s, %s]({{.Codec}}, &{{.Field}})", outerType, innerType),
				fmt.Sprintf("DecodeArrayOfStaticBytes[%s, %s]({{.Codec}}, &{{.Field}})", outerType, innerType),
				[]int{outerSize, innerSize},
			}, nil
		default:
			return nil, fmt.Errorf("unsupported array-of-array item basic type: %s", typ)
		}
	default:
		return nil, fmt.Errorf("unsupported array-of-array item type: %s", typ)
	}
}

func (p *parseContext) resolveSliceOpset(typ types.Type, tags *sizeTag) (opset, error) {
	// Sanity check a few tag constraints relevant for all slice types
	if tags == nil {
		return nil, fmt.Errorf("slice type requires ssz tags")
	}
	switch typ := typ.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Byte:
			// Slice of bytes. If we have ssz-size, it's a static slice
			if len(tags.size) > 0 {
				if (len(tags.size) != 1 && len(tags.size) != 2) ||
					(len(tags.size) == 2 && tags.size[1] != 1) {
					return nil, fmt.Errorf("static slice of byte basic type tag conflict: needs [N] or [N, 1] tag, has %v", tags.size)
				}
				if len(tags.limit) > 0 {
					return nil, fmt.Errorf("static slice of byte basic type cannot have ssz-max tag")
				}
				return &opsetStatic{
					"DefineCheckedStaticBytes({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
					"EncodeCheckedStaticBytes({{.Codec}}, &{{.Field}})",
					"DecodeCheckedStaticBytes({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
					[]int{tags.size[0]},
				}, nil
			}
			// Not a static slice of bytes, we need to pull ssz-max for the limits
			if tags.limit == nil {
				return nil, fmt.Errorf("dynamic slice of byte basic type requires ssz-max tag")
			}
			if len(tags.limit) != 1 {
				return nil, fmt.Errorf("dynamic slice of byte basic type tag conflict: needs [N] tag, has %v", tags.limit)
			}
			return &opsetDynamic{
				"SizeDynamicBytes({{.Field}})",
				"DefineDynamicBytesOffset({{.Codec}}, &{{.Field}})",
				"DefineDynamicBytesContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"EncodeDynamicBytesOffset({{.Codec}}, &{{.Field}})",
				"EncodeDynamicBytesContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"DecodeDynamicBytesOffset({{.Codec}}, &{{.Field}})",
				"DecodeDynamicBytesContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				[]int{0}, tags.limit,
			}, nil

		case types.Uint64:
			// Slice of uint64s. If we have ssz-size, it's a static slice
			if len(tags.size) > 0 {
				if (len(tags.size) != 1 && len(tags.size) != 2) ||
					(len(tags.size) == 2 && tags.size[1] != 8) {
					return nil, fmt.Errorf("static slice of uint64 basic type tag conflict: needs [N] or [N, 8] tag, has %v", tags.size)
				}
				if len(tags.limit) > 0 {
					return nil, fmt.Errorf("static slice of uint64 basic type cannot have ssz-max tag")
				}
				return &opsetStatic{
					"DefineCheckedStaticUint64({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
					"EncodeCheckedStaticUint64({{.Codec}}, &{{.Field}})",
					"DecodeCheckedStaticUint64({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
					[]int{tags.size[0]},
				}, nil
			}
			// Not a static slice of bytes, we need to pull ssz-max for the limits
			if tags.limit == nil {
				return nil, fmt.Errorf("dynamic slice of uint64 basic type requires ssz-max tag")
			}
			if len(tags.limit) != 1 {
				return nil, fmt.Errorf("dynamic slice of uint64 basic type tag conflict: needs [N] tag, has %v", tags.limit)
			}
			return &opsetDynamic{
				"SizeSliceOfUint64s({{.Field}})",
				"DefineSliceOfUint64sOffset({{.Codec}}, &{{.Field}})",
				"DefineSliceOfUint64sContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"EncodeSliceOfUint64sOffset({{.Codec}}, &{{.Field}})",
				"EncodeSliceOfUint64sContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"DecodeSliceOfUint64sOffset({{.Codec}}, &{{.Field}})",
				"DecodeSliceOfUint64sContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				nil, tags.limit,
			}, nil

		default:
			return nil, fmt.Errorf("unsupported slice item basic type: %s", typ)
		}
	case *types.Pointer:
		if types.Implements(typ, p.staticObjectIface) {
			if len(tags.size) > 0 {
				return nil, fmt.Errorf("static slice of static objects not yet implemented")
			}
			if len(tags.limit) != 1 {
				return nil, fmt.Errorf("dynamic slice of static objects type tag conflict: needs [N] tag, has %v", tags.limit)
			}
			return &opsetDynamic{
				"SizeSliceOfStaticObjects({{.Field}})",
				"DefineSliceOfStaticObjectsOffset({{.Codec}}, &{{.Field}})",
				"DefineSliceOfStaticObjectsContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"EncodeSliceOfStaticObjectsOffset({{.Codec}}, &{{.Field}})",
				"EncodeSliceOfStaticObjectsContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"DecodeSliceOfStaticObjectsOffset({{.Codec}}, &{{.Field}})",
				"DecodeSliceOfStaticObjectsContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				nil, tags.limit,
			}, nil
		}
		if types.Implements(typ, p.dynamicObjectIface) {
			if len(tags.size) > 0 {
				return nil, fmt.Errorf("static slice of dynamic objects not yet implemented")
			}
			if len(tags.limit) != 1 {
				return nil, fmt.Errorf("dynamic slice of dynamic objects type tag conflict: needs [N] tag, has %v", tags.limit)
			}
			return &opsetDynamic{
				"SizeSliceOfDynamicObjects({{.Field}})",
				"DefineSliceOfDynamicObjectsOffset({{.Codec}}, &{{.Field}})",
				"DefineSliceOfDynamicObjectsContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"EncodeSliceOfDynamicObjectsOffset({{.Codec}}, &{{.Field}})",
				"EncodeSliceOfDynamicObjectsContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"DecodeSliceOfDynamicObjectsOffset({{.Codec}}, &{{.Field}})",
				"DecodeSliceOfDynamicObjectsContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				nil, tags.limit,
			}, nil

		}
		return nil, fmt.Errorf("unsupported pointer slice item type %s", typ.String())

	case *types.Array:
		return p.resolveSliceOfArrayOpset(typ.Elem(), int(typ.Len()), tags)

	case *types.Slice:
		return p.resolveSliceOfSliceOpset(typ.Elem(), tags)

	case *types.Named:
		return p.resolveSliceOpset(typ.Underlying(), tags)

	default:
		return nil, fmt.Errorf("unsupported slice item type: %s", typ)
	}
}

func (p *parseContext) resolveSliceOfArrayOpset(typ types.Type, innerSize int, tags *sizeTag) (opset, error) {
	switch typ := typ.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Byte:
			// Slice of array of bytes. If we have ssz-size, it's a static slice.
			if len(tags.size) > 0 {
				if (len(tags.size) != 1 && len(tags.size) != 2) ||
					(len(tags.size) == 2 && tags.size[1] != innerSize) {
					return nil, fmt.Errorf("static slice of array of byte basic type tag conflict: needs [N] or [N, %d] tag, has %v", innerSize, tags.size)
				}
				if len(tags.limit) > 0 {
					return nil, fmt.Errorf("static slice of array of byte basic type cannot have ssz-max tag")
				}
				return &opsetStatic{
					"DefineCheckedArrayOfStaticBytes({{.Codec}}, &{{.Field}}, {{.MaxItems}})",
					"EncodeCheckedArrayOfStaticBytes({{.Codec}}, &{{.Field}})",
					"DecodeCheckedArrayOfStaticBytes({{.Codec}}, &{{.Field}}, {{.MaxItems}})",
					[]int{tags.size[0], innerSize},
				}, nil
			}
			// Not a static slice of array of bytes, we need to pull ssz-max for the limits
			if tags.limit == nil {
				return nil, fmt.Errorf("dynamic slice of array of byte basic type requires ssz-max tag")
			}
			if len(tags.limit) != 1 {
				return nil, fmt.Errorf("dynamic slice of array of byte basic type tag conflict: needs [N] tag, has %v", tags.limit)
			}
			return &opsetDynamic{
				"SizeSliceOfStaticBytes({{.Field}})",
				"DefineSliceOfStaticBytesOffset({{.Codec}}, &{{.Field}})",
				"DefineSliceOfStaticBytesContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"EncodeSliceOfStaticBytesOffset({{.Codec}}, &{{.Field}})",
				"EncodeSliceOfStaticBytesContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				"DecodeSliceOfStaticBytesOffset({{.Codec}}, &{{.Field}})",
				"DecodeSliceOfStaticBytesContent({{.Codec}}, &{{.Field}}, {{.MaxSize}})",
				nil, tags.limit,
			}, nil
		default:
			return nil, fmt.Errorf("unsupported array-of-array item basic type: %s", typ)
		}
	default:
		return nil, fmt.Errorf("unsupported array-of-array item type: %s", typ)
	}
}

func (p *parseContext) resolveSliceOfSliceOpset(typ types.Type, tags *sizeTag) (*opsetDynamic, error) {
	switch typ := typ.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Byte:
			// Slice of slice of bytes. At this point we have 2D possibilities of
			// ssz-size and ssz-max combinations, each resulting in a different
			// call that we have to make. Reject any conflicts in the tags, after
			// which assemble the required combo.
			switch {
			case len(tags.size) > 0 && len(tags.limit) == 0:
				return nil, fmt.Errorf("static slice of static slice of bytes not implemented yet")

			case len(tags.size) == 0 && len(tags.limit) > 0:
				if len(tags.limit) != 2 {
					return nil, fmt.Errorf("dynamic slice of dynamic slice of byte basic type tag conflict: needs [N, M] ssz-max tag, has %v", tags.limit)
				}
				return &opsetDynamic{
					"SizeSliceOfDynamicBytes({{.Field}})",
					"DefineSliceOfDynamicBytesOffset({{.Codec}}, &{{.Field}})",
					"DefineSliceOfDynamicBytesContent({{.Codec}}, &{{.Field}}, {{.MaxItems}}, {{.MaxSize}})",
					"EncodeSliceOfDynamicBytesOffset({{.Codec}}, &{{.Field}})",
					"EncodeSliceOfDynamicBytesContent({{.Codec}}, &{{.Field}}, {{.MaxItems}}, {{.MaxSize}})",
					"DecodeSliceOfDynamicBytesOffset({{.Codec}}, &{{.Field}})",
					"DecodeSliceOfDynamicBytesContent({{.Codec}}, &{{.Field}}, {{.MaxItems}}, {{.MaxSize}})",
					nil, tags.limit,
				}, nil

			default:
				return nil, fmt.Errorf("not implemented yet")
			}
		default:
			return nil, fmt.Errorf("unsupported slice-of-slice item basic type: %s", typ)
		}
	default:
		return nil, fmt.Errorf("unsupported slice-of-slice item type: %s", typ)
	}
}

func (p *parseContext) resolvePointerOpset(typ *types.Pointer, tags *sizeTag) (opset, error) {
	if isUint256(typ.Elem()) {
		if tags != nil {
			if tags.limit != nil {
				return nil, fmt.Errorf("uint256 basic type cannot have ssz-max tag")
			}
			if len(tags.size) != 1 || tags.size[0] != 32 {
				return nil, fmt.Errorf("uint256 basic type tag conflict: filed is [32] bytes, tag wants %v", tags.size)
			}
		}
		return &opsetStatic{
			"DefineUint256({{.Codec}}, &{{.Field}})",
			"EncodeUint256({{.Codec}}, &{{.Field}})",
			"DecodeUint256({{.Codec}}, &{{.Field}})",
			[]int{32},
		}, nil
	}
	if types.Implements(typ, p.staticObjectIface) {
		if tags != nil {
			return nil, fmt.Errorf("static object type cannot have any ssz tags")
		}
		return &opsetStatic{
			"DefineStaticObject({{.Codec}}, &{{.Field}})",
			"EncodeStaticObject({{.Codec}}, &{{.Field}})",
			"DecodeStaticObject({{.Codec}}, &{{.Field}})",
			nil,
		}, nil
	}
	if types.Implements(typ, p.dynamicObjectIface) {
		if tags != nil {
			return nil, fmt.Errorf("dynamic object type cannot have any ssz tags")
		}
		return &opsetDynamic{
			"SizeDynamicObject({{.Field}})",
			"DefineDynamicObjectOffset({{.Codec}}, &{{.Field}})",
			"DefineDynamicObjectContent({{.Codec}}, &{{.Field}})",
			"EncodeDynamicObjectOffset({{.Codec}}, &{{.Field}})",
			"EncodeDynamicObjectContent({{.Codec}}, &{{.Field}})",
			"DecodeDynamicObjectOffset({{.Codec}}, &{{.Field}})",
			"DecodeDynamicObjectContent({{.Codec}}, &{{.Field}})",
			nil, nil,
		}, nil
	}
	return nil, fmt.Errorf("unsupported pointer type %s", typ.String())
}
