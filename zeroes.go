// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import (
	"reflect"
	"sync"
)

// zeroCache contains zero-values for dynamic objects that got hit during codec
// operations. This is a global sync map, meaning it will be slow to access, but
// encoding/hashing zero values should not happen in production code, it's more
// of a sanity thing to handle weird corner-cases without blowing up.
var zeroCache = new(sync.Map)

// zeroValueStatic retrieves a previously created (or creates one on the fly)
// zero value for a static object to support operating on half-initialized
// objects (useful for tests mainly, but can also avoid crashes in case of bad
// calling parameters).
func zeroValueStatic[T newableStaticObject[U], U any]() T {
	kind := reflect.TypeFor[U]()

	if val, ok := zeroCache.Load(kind); ok {
		return val.(T)
	}
	val := T(new(U))
	zeroCache.Store(kind, val)
	return val
}

// zeroValueDynamic retrieves a previously created (or creates one on the fly)
// zero value for a dynamic object to support operating on half-initialized
// objects (useful for tests mainly, but can also avoid crashes in case of bad
// calling parameters).
func zeroValueDynamic[T newableDynamicObject[U], U any]() T {
	kind := reflect.TypeFor[U]()

	if val, ok := zeroCache.Load(kind); ok {
		return val.(T)
	}
	val := T(new(U))
	zeroCache.Store(kind, val)
	return val
}
