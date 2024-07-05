// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"strings"
)

func pkgName(pkgPath string) string {
	index := strings.LastIndex(pkgPath, "/")
	if index == -1 {
		return pkgPath // universal package
	}
	return pkgPath[index+1:]
}
