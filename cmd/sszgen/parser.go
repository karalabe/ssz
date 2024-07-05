// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"errors"
	"go/types"
)

func parsePackage(pkg *types.Package, names []string) ([]sszType, error) {
	if len(names) == 0 {
		names = pkg.Scope().Names()
	}
	var types []sszType
	for _, name := range names {
		named, err := lookupType(pkg.Scope(), name)
		if err != nil {
			return nil, err
		}
		typ, err := buildType(nil, named, nil)
		if err != nil {
			return nil, err
		}
		types = append(types, typ)
	}
	return types, nil
}

func lookupType(scope *types.Scope, name string) (*types.Named, error) {
	obj := scope.Lookup(name)
	if obj == nil {
		return nil, errors.New("no such identifier")
	}
	typ, ok := obj.(*types.TypeName)
	if !ok {
		return nil, errors.New("not a type")
	}
	return typ.Type().(*types.Named), nil
}
