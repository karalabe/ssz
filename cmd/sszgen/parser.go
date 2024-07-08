// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"go/types"
)

// parseContext contains some helpers for interpreting generated types.
type parseContext struct {
	staticObjectIface  *types.Interface
	dynamicObjectIface *types.Interface
}

// newParseContext loads a few ssz library interfaces for the generator.
func newParseContext(library *types.Package) *parseContext {
	var (
		static  = library.Scope().Lookup("StaticObject").Type().Underlying()
		dynamic = library.Scope().Lookup("DynamicObject").Type().Underlying()
	)
	return &parseContext{
		staticObjectIface:  static.(*types.Interface),
		dynamicObjectIface: dynamic.(*types.Interface),
	}
}

// parsePackage retrieves the specified named-types from the target package and
// creates ssz containers out of them.
func (p *parseContext) parsePackage(target *types.Package, names []string) ([]*sszContainer, error) {
	// If no types were requested, parse all of them
	if len(names) == 0 {
		names = target.Scope().Names()
	}
	var containers []*sszContainer
	for _, name := range names {
		named, str, err := p.lookupStruct(target.Scope(), name)
		if err != nil {
			return nil, err
		}
		typ, err := p.makeContainer(named, str)
		if err != nil {
			return nil, err
		}
		containers = append(containers, typ)
	}
	return containers, nil
}

// lookupStruct is a small helper to check that a type name is indeed a struct
// that we can convert into an ssz type.
func (p *parseContext) lookupStruct(scope *types.Scope, name string) (*types.Named, *types.Struct, error) {
	obj := scope.Lookup(name)
	if obj == nil {
		return nil, nil, fmt.Errorf("identifier not found: %s", name)
	}
	typ, ok := obj.(*types.TypeName)
	if !ok {
		return nil, nil, fmt.Errorf("identifier not a type: %s", name)
	}
	dec, ok := typ.Type().(*types.Named)
	if !ok {
		return nil, nil, fmt.Errorf("identifier not a named type: %s", name)
	}
	str, ok := dec.Underlying().(*types.Struct)
	if !ok {
		return nil, nil, fmt.Errorf("identifier not a named struct: %s", name)
	}
	return dec, str, nil
}
