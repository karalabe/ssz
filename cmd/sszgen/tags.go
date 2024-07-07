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
)

// sizeTag describes the size restriction for types.
type sizeTag struct {
	size  int64 // 0 means the size is undefined
	limit int64 // 0 means the limit is undefined
}

func parseTag(input string) (bool, []sizeTag, error) {
	if len(input) == 0 {
		return false, nil, nil
	}
	var (
		ignore bool
		tags   []sizeTag
		setTag = func(i int, v int64, ident string) {
			if i >= len(tags) {
				tags = append(tags, make([]sizeTag, i-len(tags)+1)...)
			}
			if ident == sszMaxTagIdent {
				tags[i].limit = v
			} else {
				tags[i].size = v
			}
		}
	)
	for _, tag := range strings.Split(input, " ") {
		parts := strings.Split(tag, ":")
		if len(parts) != 2 {
			return false, nil, fmt.Errorf("invalid tag %s", tag)
		}
		ident, remain := parts[0], strings.Trim(parts[1], "\"")
		switch ident {
		case sszTagIdent:
			if remain == "-" {
				ignore = true
			}
		case sszMaxTagIdent, sszSizeTagIdent:
			parts := strings.Split(remain, ",")
			for i, p := range parts {
				if p == "?" {
					setTag(i, 0, ident)
					continue
				}
				num, err := strconv.ParseInt(p, 10, 64)
				if err != nil {
					return false, nil, err
				}
				setTag(i, num, ident)
			}
		}
	}
	return ignore, tags, nil
}
