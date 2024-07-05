// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"fmt"
	"go/types"

	"github.com/rjl493456442/sszgen/ssz"
)

type sszType interface {
	// Attributes
	fixed() bool
	fixedSize() int
	typeName() string

	// Operations
	genSize(ctx *genContext, w string, obj string) string
	genEncoder(ctx *genContext, obj string) string
	genDecoder(ctx *genContext, r string, obj string) string
}

func buildType(named *types.Named, typ types.Type, tags []sizeTag) (sszType, error) {
	switch t := typ.(type) {
	case *types.Named:
		if isBigInt(typ) {
			//return bigIntOp{}, nil
		}
		if isUint256(typ) {
			//return uint256Op{}, nil
		}
		return buildType(t, typ.Underlying(), tags)
	case *types.Basic:
		return newBasic(named, t)
	case *types.Array:
		return newVector(named, t, tags)
	case *types.Slice:
		return newList(named, t, tags)
	case *types.Pointer:
		if isBigInt(t.Elem()) {
			//return bigIntOp{pointer: true}, nil
		}
		if isUint256(t.Elem()) {
			//return uint256Op{pointer: true}, nil
		}
		return newPointer(named, t, tags)
	case *types.Struct:
		return newStruct(named, t)
	}
	return nil, fmt.Errorf("unsupported type %s", typ.String())
}

type sszBasic struct {
	basic   *types.Basic
	named   *types.Named
	size    int
	encoder string
	decoder string
}

func newBasic(named *types.Named, typ *types.Basic) (*sszBasic, error) {
	var (
		size    int
		encoder string
		decoder string
		kind    = typ.Kind()
	)
	switch {
	case kind == types.Bool:
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
		decoder = fmt.Sprintf("DecodeUint%d", size*8)
	default:
		return nil, fmt.Errorf("unsupported basic type: %s", typ.String())
	}
	return &sszBasic{
		basic:   typ,
		named:   named,
		size:    size,
		encoder: encoder,
		decoder: decoder,
	}, nil
}

func (b *sszBasic) fixed() bool {
	return true
}

func (b *sszBasic) fixedSize() int {
	return b.size
}

func (b *sszBasic) typeName() string {
	return b.basic.String()
}

func (b *sszBasic) genSize(ctx *genContext, w string, obj string) string {
	return fmt.Sprintf("%s += %d\n", w, b.size)
}

func (b *sszBasic) genEncoder(ctx *genContext, obj string) string {
	ctx.addImport(pkgPath, "")
	if b.named != nil {
		obj = fmt.Sprintf("%s(%s)", b.typeName(), obj) // explicit type conversion
	}
	return fmt.Sprintf("%s(w, %s)\n", ctx.qualifier(pkgPath, b.encoder), obj)
}

func (b *sszBasic) genDecoder(ctx *genContext, r string, obj string) string {
	var (
		v   = ctx.tmpVar("v")
		err = ctx.tmpVar("e")
		buf bytes.Buffer
	)
	ctx.addImport(pkgPath, "")
	fmt.Fprintf(&buf, "%s, %s := %s(%s)\n", v, err, ctx.qualifier(pkgPath, b.decoder), r)
	fmt.Fprintf(&buf, "if %s != nil {\n", err)
	fmt.Fprintf(&buf, "return %s\n", err)
	fmt.Fprint(&buf, "}\n")
	if b.named != nil {
		v = fmt.Sprintf("%s(%s)", b.named.Obj().Name(), v) // explicit type conversion
	}
	fmt.Fprintf(&buf, "%s = %s\n", obj, v)
	return buf.String()
}

type sszVector struct {
	array   *types.Array
	named   *types.Named
	elem    sszType
	len     int64
	tag     sizeTag
	encoder string
	decoder string
}

func newVector(named *types.Named, typ *types.Array, tags []sizeTag) (*sszVector, error) {
	var (
		tag    sizeTag
		remain []sizeTag
	)
	if len(tags) > 0 {
		tag, remain = tags[0], tags[1:]
	}
	if tag.size != 0 && tag.size != typ.Len() {
		return nil, fmt.Errorf("invalid size tag, array: %d, tag %d", typ.Len(), tag.size)
	}
	if tag.limit != 0 {
		return nil, fmt.Errorf("unexpected size limit tag")
	}
	elem, err := buildType(nil, typ.Elem(), remain)
	if err != nil {
		return nil, err
	}
	var (
		encoder string
		decoder string
	)
	if b, ok := elem.(*sszBasic); ok {
		encoder = fmt.Sprintf("%ss", b.encoder)
		decoder = fmt.Sprintf("%ss", b.decoder)
	}
	return &sszVector{
		array:   typ,
		named:   named,
		elem:    elem,
		len:     typ.Len(),
		tag:     tag,
		encoder: encoder,
		decoder: decoder,
	}, nil
}

func (v *sszVector) fixed() bool {
	return v.elem.fixed()
}

func (v *sszVector) fixedSize() int {
	if v.fixed() {
		return int(v.len) * v.elem.fixedSize()
	}
	return ssz.BytesPerLengthOffset
}

func (v *sszVector) typeName() string {
	return v.array.String()
}

func (v *sszVector) genSize(ctx *genContext, w string, obj string) string {
	if v.elem.fixed() {
		return fmt.Sprintf("%s += %d\n", w, int(v.len)*v.elem.fixedSize())
	}
	var (
		b   bytes.Buffer
		vid = ctx.tmpVar("v")
	)
	fmt.Fprintf(&b, "for _, %s := range %s {\n", vid, obj)
	fmt.Fprintf(&b, "%s", v.elem.genSize(ctx, w, vid))
	fmt.Fprint(&b, "}\n")
	return b.String()
}

func (v *sszVector) genEncoder(ctx *genContext, obj string) string {
	if v.encoder != "" {
		return fmt.Sprintf("%s(w, %s[:])\n", ctx.qualifier(pkgPath, v.encoder), obj)
	}
	var b bytes.Buffer
	if !v.elem.fixed() {
		offset := ctx.tmpVar("o")
		fmt.Fprintf(&b, "%s := len(%s)*4\n", offset, obj)

		vid := ctx.tmpVar("v")
		fmt.Fprintf(&b, "for _, %s := range %s {\n", vid, obj)
		fmt.Fprintf(&b, "%s(w, uint32(%s))\n", ctx.qualifier(pkgPath, "EncodeUint32"), offset)
		fmt.Fprintf(&b, "%s", v.elem.genSize(ctx, offset, vid))
		fmt.Fprint(&b, "}\n")
	}
	vid := ctx.tmpVar("v")
	fmt.Fprintf(&b, "for _, %s := range %s {\n", vid, obj)
	fmt.Fprintf(&b, "%s", v.elem.genEncoder(ctx, vid))
	fmt.Fprint(&b, "}\n")
	return b.String()
}

func (v *sszVector) genDecoder(ctx *genContext, r string, obj string) string {
	var b bytes.Buffer
	if v.decoder != "" {
		var (
			vn  = ctx.tmpVar("v")
			err = ctx.tmpVar("e")
		)
		fmt.Fprintf(&b, "%s, %s := %s(%s, %d)\n", vn, err, ctx.qualifier(pkgPath, v.decoder), r, v.len)
		fmt.Fprintf(&b, "if %s != nil {\n", err)
		fmt.Fprintf(&b, "return %s\n", err)
		fmt.Fprint(&b, "}\n")
		fmt.Fprintf(&b, "%s = %s(%s)\n", obj, v.typeName(), vn)
		return b.String()
	}
	if v.elem.fixed() {
		var cnt = ctx.tmpVar("i")
		fmt.Fprintf(&b, "for %s := 0; %s < %d; %s += 1 {\n", cnt, cnt, v.len, cnt)
		fmt.Fprintf(&b, "%s", v.elem.genDecoder(ctx, r, fmt.Sprintf("%s[%s]", obj, cnt)))
		fmt.Fprint(&b, "}\n") // curly brace for loop
		return b.String()
	}
	// Decode offsets
	cnt := ctx.tmpVar("i")
	fmt.Fprintf(&b, "for %s := 0; %s < %d; %s += 1 {\n", cnt, cnt, v.len, cnt)
	fmt.Fprintf(&b, "if err := %s.DecodeOffset(); err != nil{\n", r)
	fmt.Fprintf(&b, "return err\n")
	fmt.Fprint(&b, "}\n") // curly brace for if
	fmt.Fprint(&b, "}\n") // curly brace for loop

	// Decode elements
	fmt.Fprintf(&b, "for %s := 0; %s < %d; %s += 1 {\n", cnt, cnt, v.len, cnt)
	wrapList(ctx, r, &b, func() {
		fmt.Fprintf(&b, "%s", v.elem.genDecoder(ctx, r, fmt.Sprintf("%s[%s]", obj, cnt)))
	})
	fmt.Fprint(&b, "}\n") // curly brace for loop
	return b.String()
}

type sszList struct {
	slice   *types.Slice
	named   *types.Named
	elem    sszType
	tag     sizeTag
	encoder string
	decoder string
}

func newList(named *types.Named, slice *types.Slice, tags []sizeTag) (*sszList, error) {
	var (
		tag    sizeTag
		remain []sizeTag
	)
	if len(tags) > 0 {
		tag, remain = tags[0], tags[1:]
	}
	elem, err := buildType(nil, slice.Elem(), remain)
	if err != nil {
		return nil, err
	}
	var (
		encoder string
		decoder string
	)
	if b, ok := elem.(*sszBasic); ok {
		encoder = fmt.Sprintf("%ss", b.encoder)
		decoder = fmt.Sprintf("%ss", b.decoder)
	}
	return &sszList{
		slice:   slice,
		named:   named,
		elem:    elem,
		tag:     tag,
		encoder: encoder,
		decoder: decoder,
	}, nil
}

func (l *sszList) fixed() bool {
	if l.tag.size != 0 {
		return l.elem.fixed()
	}
	return false
}

func (l *sszList) fixedSize() int {
	if l.fixed() {
		return int(l.tag.size) * l.elem.fixedSize()
	}
	return ssz.BytesPerLengthOffset
}

func (l *sszList) typeName() string {
	return l.slice.String()
}

func (l *sszList) genSize(ctx *genContext, w string, obj string) string {
	if l.elem.fixed() {
		if l.elem.fixedSize() == 1 {
			return fmt.Sprintf("%s += len(%s)\n", w, obj)
		}
		return fmt.Sprintf("%s += len(%s)*%d\n", w, obj, l.elem.fixedSize())
	}
	var (
		b   bytes.Buffer
		vid = ctx.tmpVar("v")
	)
	fmt.Fprintf(&b, "for _, %s := range %s {\n", vid, obj)
	fmt.Fprintf(&b, "%s += 4\n", w)
	fmt.Fprintf(&b, "%s", l.elem.genSize(ctx, w, vid))
	fmt.Fprint(&b, "}\n")
	return b.String()
}

func (l *sszList) genEncoder(ctx *genContext, obj string) string {
	if l.encoder != "" {
		return fmt.Sprintf("%s(w, %s)\n", ctx.qualifier(pkgPath, l.encoder), obj)
	}
	var b bytes.Buffer
	if !l.elem.fixed() {
		oid := ctx.tmpVar("o")
		fmt.Fprintf(&b, "%s := len(%s)*4\n", oid, obj)

		vid := ctx.tmpVar("v")
		fmt.Fprintf(&b, "for _, %s := range %s {\n", vid, obj)
		fmt.Fprintf(&b, "%s(w, uint32(%s))\n", ctx.qualifier(pkgPath, "EncodeUint32"), oid)
		fmt.Fprintf(&b, "%s", l.elem.genSize(ctx, oid, vid))
		fmt.Fprint(&b, "}\n")
	}
	vid := ctx.tmpVar("v")
	fmt.Fprintf(&b, "for _, %s := range %s {\n", vid, obj)
	fmt.Fprintf(&b, "%s", l.elem.genEncoder(ctx, vid))
	fmt.Fprint(&b, "}\n")
	return b.String()
}

func (l *sszList) genDecoder(ctx *genContext, r string, obj string) string {
	var b bytes.Buffer
	ctx.addImport(pkgPath, "")

	if l.decoder != "" {
		var (
			v   = ctx.tmpVar("v")
			err = ctx.tmpVar("e")
		)
		fmt.Fprintf(&b, "%s, %s := %s(%s, %d)\n", v, err, ctx.qualifier(pkgPath, l.decoder), r, l.tag.size)
		fmt.Fprintf(&b, "if %s != nil {\n", err)
		fmt.Fprintf(&b, "return %s\n", err)
		fmt.Fprint(&b, "}\n")
		fmt.Fprintf(&b, "%s = %s\n", obj, v)
		return b.String()
	}
	if l.elem.fixed() {
		var cnt = ctx.tmpVar("i")
		fmt.Fprintf(&b, "for %s := 0; %s < len(%s); %s += 1 {\n", cnt, cnt, obj, cnt)
		fmt.Fprintf(&b, "%s", l.elem.genDecoder(ctx, r, fmt.Sprintf("%s[%s]", obj, cnt)))
		fmt.Fprint(&b, "}\n") // curly brace for loop
		return b.String()
	}
	// Decode length

	// length, err := s.DecodeOffset()
	// for i := 1; i < length; i++ {
	//      DecodeOffset
	// }
	// for i := 0; i < length; i++ {
	//      DecodeOffset
	// }

	cnt := ctx.tmpVar("i")
	fmt.Fprintf(&b, "for %s := 0; %s < len(%s); %s += 1 {\n", cnt, cnt, obj, cnt)
	fmt.Fprintf(&b, "if err := %s.DecodeOffset(); err != nil{\n", r)
	fmt.Fprintf(&b, "return err\n")
	fmt.Fprint(&b, "}\n") // curly brace for if
	fmt.Fprint(&b, "}\n") // curly brace for loop

	// Decode elements
	fmt.Fprintf(&b, "for %s := 0; %s < len(%s); %s += 1 {\n", cnt, cnt, obj, cnt)
	wrapList(ctx, r, &b, func() {
		fmt.Fprintf(&b, "%s", l.elem.genDecoder(ctx, r, fmt.Sprintf("%s[%s]", obj, cnt)))
	})
	fmt.Fprint(&b, "}\n") // curly brace for loop
	return b.String()
}

type sszStruct struct {
	*types.Struct
	named      *types.Named
	fields     []sszType
	fieldNames []string
}

func newStruct(named *types.Named, typ *types.Struct) (*sszStruct, error) {
	var (
		fields     []sszType
		fieldNames []string
	)
	for i := 0; i < typ.NumFields(); i++ {
		f := typ.Field(i)
		if !f.Exported() {
			continue
		}
		var (
			err     error
			ignored bool
			tags    []sizeTag
		)
		if tag := typ.Tag(i); tag != "" {
			ignored, tags, err = parseTag(tag)
			if err != nil {
				return nil, err
			}
		}
		if ignored {
			continue
		}
		field, err := buildType(nil, f.Type(), tags)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
		fieldNames = append(fieldNames, f.Name())
	}
	return &sszStruct{
		Struct:     typ,
		named:      named,
		fields:     fields,
		fieldNames: fieldNames,
	}, nil
}

func (s *sszStruct) fixed() bool {
	for _, field := range s.fields {
		if !field.fixed() {
			return false
		}
	}
	return true
}

func (s *sszStruct) fixedSize() int {
	if !s.fixed() {
		return ssz.BytesPerLengthOffset
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
			fmt.Fprintf(&b, "%s(w, uint32(%s))\n", ctx.qualifier(pkgPath, "EncodeUint32"), oid)
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

type sszPointer struct {
	*types.Pointer
	named *types.Named
	elem  sszType
}

func newPointer(named *types.Named, typ *types.Pointer, tags []sizeTag) (*sszPointer, error) {
	elem, err := buildType(nil, typ.Elem(), tags)
	if err != nil {
		return nil, err
	}
	return &sszPointer{
		Pointer: typ,
		named:   named,
		elem:    elem,
	}, nil
}

func (p *sszPointer) typeName() string {
	return fmt.Sprintf("*%s", p.elem.typeName())
}

func (p *sszPointer) fixed() bool {
	return p.elem.fixed()
}

func (p *sszPointer) fixedSize() int {
	return p.elem.fixedSize()
}

func (p *sszPointer) genSize(ctx *genContext, w string, obj string) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "if %s == nil {\n", obj)
	fmt.Fprintf(&b, "%s = new(%s)\n", obj, p.elem.typeName())
	fmt.Fprint(&b, "}\n")
	fmt.Fprintf(&b, "%s", p.elem.genSize(ctx, w, obj))
	return b.String()
}

func (p *sszPointer) genEncoder(ctx *genContext, obj string) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "if %s == nil {\n", obj)
	fmt.Fprintf(&b, "%s = new(%s)\n", obj, p.elem.typeName())
	fmt.Fprint(&b, "}\n")
	fmt.Fprintf(&b, "%s", p.elem.genEncoder(ctx, obj))
	return b.String()
}

func (p *sszPointer) genDecoder(ctx *genContext, r string, obj string) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "if %s == nil {\n", obj)
	fmt.Fprintf(&b, "%s = new(%s)\n", obj, p.elem.typeName())
	fmt.Fprint(&b, "}\n")
	fmt.Fprintf(&b, "%s", p.elem.genDecoder(ctx, r, obj))
	return b.String()
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
