// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"go/types"
)

type sszContainer struct {
	*types.Struct
	named  *types.Named
	static bool
	fields []string
	opsets []opset
}

func newContainer(named *types.Named, typ *types.Struct) (*sszContainer, error) {
	var (
		static = true
		fields []string
		opsets []opset
	)
	// Iterate over all the fields of the struct
	for i := 0; i < typ.NumFields(); i++ {
		// Skip private fields, and skip ignored ssz fields
		f := typ.Field(i)
		if !f.Exported() {
			continue
		}
		ignore, tags, err := parseTag(typ.Tag(i))
		if err != nil {
			return nil, err
		}
		if ignore {
			continue
		}
		// Required field found, validate type with tag content
		opset, err := validateField(f.Type(), tags)
		if err != nil {
			return nil, fmt.Errorf("failed to validate field %s.%s: %v", named.Obj().Name(), f.Name(), err)
		}
		if _, ok := (opset).(*opsetDynamic); ok {
			static = false
		}
		fields = append(fields, f.Name())
		opsets = append(opsets, opset)
	}
	return &sszContainer{
		Struct: typ,
		named:  named,
		static: static,
		fields: fields,
		opsets: opsets,
	}, nil
}

// validateContainerField compares the type of the field to the provided tags and
// returns whether there's a collision between them, or if more tags are needed to
// fully derive the size.
func validateField(typ types.Type, tags []sizeTag) (opset, error) {
	switch t := typ.(type) {
	case *types.Named:
		return validateField(t.Underlying(), tags)

	case *types.Basic:
		return resolveBasicOpset(t)

	case *types.Array:
		return resolveArrayOpset(t.Elem(), int(t.Len()))

	case *types.Slice:
		return resolveSliceOpset(t.Elem())

	case *types.Pointer:
		if isUint256(t.Elem()) {
			return resolveUint256Opset(), nil
		}
		return nil, fmt.Errorf("unsupported pointer type %s", typ.String())
		/*case *types.Struct:
		return newStruct(named, t)*/
	}
	return nil, fmt.Errorf("unsupported type %s", typ.String())
}

func validateBasic(typ *types.Basic, tags []sizeTag) error {
	kind := typ.Kind()
	switch {
	/*case kind == types.Bool:
		size = 1
		encoder = "EncodeBool"
		decoder = "DecodeBool"
	case kind == types.Uint8:
		size = 1
		encoder = "EncodeByte"
		decoder = "DecodeByte"
	case kind > types.Uint8 && kind <= types.Uint64:
		size = 1 << (kind - types.Uint8)
		encoder = fmt.Sprintf("EncodeUint%d", size*8)
		decoder = fmt.Sprintf("DecodeUint%d", size*8)*/
	default:
		return fmt.Errorf("unsupported basic type: %s", kind)
	}
}

/*
func (s *sszStruct) fixed() bool {
	for _, tags := range s.fieldTags {
		for _, tag := range tags {
			if tag.limit > 0 {
				return false
			}
		}
	}
	return true
}

func (s *sszStruct) fixedSize() int {
	if !s.fixed() {
		return bytesPerLengthOffset
	}
	var size int
	for _, field := range s.fields {
		size += field.fixedSize()
	}
	return size
}

func (s *sszStruct) typeName() string {
	return s.named.Obj().Name()
}

func (s *sszStruct) genSize(ctx *genContext, w string, obj string) string {
	if !ctx.topType {
		return fmt.Sprintf("%s += %s.SizeSSZ()\n", w, obj)
	}
	ctx.topType = false

	var b bytes.Buffer
	var fixedSize int
	for _, field := range s.fields {
		fixedSize += field.fixedSize()
	}
	fmt.Fprintf(&b, "%s := %d\n", w, fixedSize)

	for i, field := range s.fields {
		if field.fixed() {
			continue
		}
		fmt.Fprintf(&b, "%s", field.genSize(ctx, w, fmt.Sprintf("%s.%s", obj, s.fieldNames[i])))
	}
	return b.String()
}

func (s *sszStruct) genEncoder(ctx *genContext, obj string) string {
	var b bytes.Buffer
	if !ctx.topType {
		fmt.Fprintf(&b, "if err := %s.MarshalSSZTo(w); err != nil {\n", obj)
		fmt.Fprint(&b, "return err\n")
		fmt.Fprint(&b, "}\n")
		return b.String()
	}
	ctx.topType = false

	var oid string
	if !s.fixed() {
		var offset int
		for _, field := range s.fields {
			offset += field.fixedSize()
		}
		oid = ctx.tmpVar("o")
		fmt.Fprintf(&b, "%s := %d\n", oid, offset)
	}
	for i, field := range s.fields {
		if field.fixed() {
			fmt.Fprintf(&b, "%s", field.genEncoder(ctx, fmt.Sprintf("%s.%s", obj, s.fieldNames[i])))
		} else {
			fmt.Fprintf(&b, "%s(w, uint32(%s))\n", ctx.qualifier(sszPkgPath, "EncodeUint32"), oid)
			fmt.Fprintf(&b, "%s", field.genSize(ctx, oid, fmt.Sprintf("%s.%s", obj, s.fieldNames[i])))
		}
	}
	for i, field := range s.fields {
		if field.fixed() {
			continue
		}
		fmt.Fprintf(&b, "%s", field.genEncoder(ctx, fmt.Sprintf("%s.%s", obj, s.fieldNames[i])))
	}
	return b.String()
}

func (s *sszStruct) genDecoder(ctx *genContext, r string, obj string) string {
	var b bytes.Buffer
	if !ctx.topType {
		fmt.Fprintf(&b, "if err := %s.UnmarshalSSZ(%s); err != nil {\n", obj, r)
		fmt.Fprint(&b, "return err\n")
		fmt.Fprint(&b, "}\n")
		return b.String()
	}
	ctx.topType = false

	for i, field := range s.fields {
		if field.fixed() {
			fmt.Fprintf(&b, "%s", field.genDecoder(ctx, r, fmt.Sprintf("%s.%s", obj, s.fieldNames[i])))
		} else {
			decodeOffset(ctx, r, &b)
		}
	}
	for i, field := range s.fields {
		if field.fixed() {
			continue
		}
		wrapList(ctx, r, &b, func() {
			fmt.Fprintf(&b, "%s", field.genDecoder(ctx, r, fmt.Sprintf("%s.%s", obj, s.fieldNames[i])))
		})
	}
	return b.String()
}



func wrapList(ctx *genContext, r string, b *bytes.Buffer, fn func()) {
	err := ctx.tmpVar("e")
	fmt.Fprintf(b, "%s := %s.BlockStart()\n", err, r)
	fmt.Fprintf(b, "if %s != nil {\n", err)
	fmt.Fprintf(b, "return %s\n", err)
	fmt.Fprint(b, "}\n")
	fn()
	fmt.Fprintf(b, "%s = %s.BlockEnd()\n", err, r)
	fmt.Fprintf(b, "if %s != nil {\n", err)
	fmt.Fprintf(b, "return %s\n", err)
	fmt.Fprint(b, "}\n")
}

func decodeOffset(ctx *genContext, r string, b *bytes.Buffer) {
	err := ctx.tmpVar("e")
	fmt.Fprintf(b, "if %s := %s.DecodeOffset(); %s != nil {\n", err, r, err)
	fmt.Fprintf(b, "return %s\n", err)
	fmt.Fprint(b, "}\n")
}

*/

// isBigInt checks whether 'typ' is "math/big".Int.
func isBigInt(typ types.Type) bool {
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	name := named.Obj()
	return name.Pkg().Path() == "math/big" && name.Name() == "Int"
}

// isUint256 checks whether 'typ' is "github.com/holiman/uint256".Int.
func isUint256(typ types.Type) bool {
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	name := named.Obj()
	return name.Pkg().Path() == "github.com/holiman/uint256" && name.Name() == "Int"
}
