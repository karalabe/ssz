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
	types  []types.Type
	opsets []opset
}

// makeContainer iterates over the fields of the struct and attempt to match each
// field with an opset for encoding/decoding ssz.
func (p *parseContext) makeContainer(named *types.Named, typ *types.Struct) (*sszContainer, error) {
	var (
		static = true
		fields []string
		types  []types.Type
		opsets []opset
	)
	// Iterate over all the fields of the struct
	for i := 0; i < typ.NumFields(); i++ {
		// Skip private fields, and skip ignored ssz fields
		f := typ.Field(i)
		if !f.Exported() {
			continue
		}
		ignore, tags, err := parseTags(typ.Tag(i))
		if err != nil {
			return nil, err
		}
		if ignore {
			continue
		}
		// Required field found, validate type with tag content
		opset, err := p.resolveOpset(f.Type(), tags)
		if err != nil {
			return nil, fmt.Errorf("failed to validate field %s.%s: %v", named.Obj().Name(), f.Name(), err)
		}
		if _, ok := (opset).(*opsetDynamic); ok {
			static = false
		}
		fields = append(fields, f.Name())
		types = append(types, f.Type())
		opsets = append(opsets, opset)
	}
	return &sszContainer{
		Struct: typ,
		named:  named,
		static: static,
		fields: fields,
		types:  types,
		opsets: opsets,
	}, nil
}

// resolveOpset compares the type of the field to the provided tags and returns
// whether there's a collision between them, or if more tags are needed to fully
// derive the size. If the type/tags are in sync and well-defined, an opset will
// be returned that the generator can use to create the code.
func (p *parseContext) resolveOpset(typ types.Type, tags *sizeTag) (opset, error) {
	switch t := typ.(type) {
	case *types.Named:
		if isBitlist(typ) {
			return p.resolveBitlistOpset(tags)
		}
		return p.resolveOpset(t.Underlying(), tags)

	case *types.Basic:
		return p.resolveBasicOpset(t, tags)

	case *types.Array:
		return p.resolveArrayOpset(t.Elem(), int(t.Len()), tags)

	case *types.Slice:
		return p.resolveSliceOpset(t.Elem(), tags)

	case *types.Pointer:
		return p.resolvePointerOpset(t, tags)
	}
	return nil, fmt.Errorf("unsupported type %s", typ.String())
}

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

// isBitlist checks whether 'typ' is "github.com/prysmaticlabs/go-bitfield".Bitlist.
func isBitlist(typ types.Type) bool {
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	name := named.Obj()
	return name.Pkg().Path() == "github.com/prysmaticlabs/go-bitfield" && name.Name() == "Bitlist"
}
