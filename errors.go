// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

import "errors"

// ErrFirstOffsetMismatch is returned when parsing dynamic types and the first
// offset (which is supposed to signal the start of the dynamic area) does not
// match with the computed fixed area size.
var ErrFirstOffsetMismatch = errors.New("ssz: first offset mismatch")

// ErrBadOffsetProgression is returned when an offset is parsed, and is smaller
// than a previously seen offset (meaning negative dynamic data size).
var ErrBadOffsetProgression = errors.New("ssz: offset smaller than previous")

// ErrOffsetBeyondCapacity is returned when an offset is parsed, and is larger
// than the total capacity allowed by the decoder (i.e. message size)
var ErrOffsetBeyondCapacity = errors.New("ssz: offset beyond capacity")

// ErrMaxLengthExceeded is returned when the size calculated for a dynamic type
// is larger than permitted.
var ErrMaxLengthExceeded = errors.New("ssz: maximum item size exceeded")

// ErrMaxItemsExceeded is returned when the number of items in a dynamic list
// type is later than permitted.
var ErrMaxItemsExceeded = errors.New("ssz: maximum item count exceeded")

// ErrShortCounterOffset is returned if a counter offset it attempted to be read
// but there are fewer bytes available on the stream.
var ErrShortCounterOffset = errors.New("ssz: insufficient data for 4-byte counter offset")

// ErrBadCounterOffset is returned when a list of offsets are consumed and the
// first offset is not a multiple of 4-bytes.
var ErrBadCounterOffset = errors.New("ssz: counter offset not multiple of 4-bytes")

// ErrDynamicStaticsIndivisible is returned when a list of static objects is to
// be decoded, but the list's total length is not divisible by the item size.
var ErrDynamicStaticsIndivisible = errors.New("ssz: list of fixed objects not divisible")
