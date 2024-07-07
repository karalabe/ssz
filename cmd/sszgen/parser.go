// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"go/types"
)

func parsePackage(pkg *types.Package, names []string) ([]*sszContainer, error) {
	if len(names) == 0 {
		names = pkg.Scope().Names()
	}
	var conts []*sszContainer
	for _, name := range names {
		named, str, err := lookupStruct(pkg.Scope(), name)
		if err != nil {
			return nil, err
		}
		typ, err := newContainer(named, str)
		if err != nil {
			return nil, err
		}
		conts = append(conts, typ)
	}
	return conts, nil
}

func lookupStruct(scope *types.Scope, name string) (*types.Named, *types.Struct, error) {
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
