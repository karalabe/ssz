// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	sszTagIdent     = "ssz"
	sszSizeTagIdent = "ssz-size"
	sszMaxTagIdent  = "ssz-max"
	sszForkTagIdent = "ssz-fork"
)

// sizeTag describes the restriction for types.
type sizeTag struct {
	bits  bool  // whether the sizes are bits instead of bytes
	size  []int // 0 means the size for that dimension is undefined
	limit []int // 0 means the limit for that dimension is undefined
}

func parseTags(input string) (bool, *sizeTag, string, error) {
	if len(input) == 0 {
		return false, nil, "", nil
	}
	var (
		ignore bool
		tags   sizeTag
		fork   string

		setTag = func(v int, ident string) {
			if ident == sszMaxTagIdent {
				tags.limit = append(tags.limit, v)
			} else {
				tags.size = append(tags.size, v)
			}
		}
	)
	for _, tag := range strings.Fields(input) {
		parts := strings.Split(tag, ":")
		if len(parts) != 2 {
			return false, nil, "", fmt.Errorf("invalid tag %s", tag)
		}
		ident, remain := parts[0], strings.Trim(parts[1], "\"")
		switch ident {
		case sszTagIdent:
			if remain == "-" {
				ignore = true
			} else if remain == "bits" {
				tags.bits = true
			}
		case sszMaxTagIdent, sszSizeTagIdent:
			parts := strings.Split(remain, ",")
			for _, p := range parts {
				if p == "?" {
					setTag(0, ident)
					continue
				}
				num, err := strconv.ParseInt(p, 10, 64)
				if err != nil {
					return false, nil, "", err
				}
				setTag(int(num), ident)
			}
		case sszForkTagIdent:
			var negate bool
			if remain[0] == '!' {
				negate = true
				remain = remain[1:]
			}
			if enum, ok := forkMapping[remain]; !ok {
				return ignore, nil, "", fmt.Errorf("invalid fork tag %s", tag)
			} else {
				fork = enum
				if negate {
					fork = "!" + fork
				}
			}
		}
	}
	if tags.size == nil && tags.limit == nil {
		return ignore, nil, fork, nil
	}
	return ignore, &tags, fork, nil
}
