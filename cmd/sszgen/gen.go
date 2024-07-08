// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"fmt"
	"go/types"
	"html/template"
	"math"
	"sort"
)

const (
	offsetBytes = 4
	sszPkgPath  = "github.com/karalabe/ssz"
)

type genContext struct {
	topType bool
	pkg     *types.Package
	imports map[string]string
	nvar    int
}

func newGenContext(pkg *types.Package) *genContext {
	return &genContext{
		pkg:     pkg,
		imports: make(map[string]string),
	}
}

func (ctx *genContext) qualifier(path string, obj string) string {
	if path == ctx.pkg.Path() {
		return obj
	}
	return fmt.Sprintf("%s.%s", pkgName(path), obj)
}

func (ctx *genContext) addImport(path string, alias string) error {
	if path == ctx.pkg.Path() {
		return nil
	}
	if n, ok := ctx.imports[path]; ok && n != alias {
		return fmt.Errorf("conflict import %s(alias: %s-%s)", path, n, alias)
	}
	ctx.imports[path] = alias
	return nil
}

func (ctx *genContext) header() []byte {
	var paths sort.StringSlice
	for path := range ctx.imports {
		paths = append(paths, path)
	}
	sort.Sort(paths)

	var b bytes.Buffer
	fmt.Fprintf(&b, "package %s\n", ctx.pkg.Name())
	if len(paths) == 0 {
		return b.Bytes()
	}
	if len(paths) == 1 {
		alias := ctx.imports[paths[0]]
		if alias == "" {
			fmt.Fprintf(&b, "import \"%s\"\n", paths[0])
		} else {
			fmt.Fprintf(&b, "import %s \"%s\"\n", alias, paths[0])
		}
		return b.Bytes()
	}
	fmt.Fprintf(&b, "import (\n")
	for _, path := range paths {
		alias := ctx.imports[path]
		if alias == "" {
			fmt.Fprintf(&b, "\"%s\"\n", path)
		} else {
			fmt.Fprintf(&b, "%s \"%s\"\n", alias, path)
		}
	}
	fmt.Fprintf(&b, ")\n")
	return b.Bytes()
}

func (ctx *genContext) tmpVar(name string) string {
	id := fmt.Sprintf("_%s%d", name, ctx.nvar)
	ctx.nvar += 1
	return id
}

func (ctx *genContext) reset() {
	ctx.nvar = 0
	ctx.topType = true
}

func generate(ctx *genContext, typ *sszContainer) ([]byte, error) {
	var codes [][]byte
	for _, fn := range []func(ctx *genContext, typ *sszContainer) ([]byte, error){
		generateSizeSSZ,
		generateDefineSSZ,
	} {
		code, err := fn(ctx, typ)
		if err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	fmt.Println(string(bytes.Join(codes, []byte("\n"))))
	return bytes.Join(codes, []byte("\n")), nil
}

func generateSizeSSZ(ctx *genContext, typ *sszContainer) ([]byte, error) {
	var b bytes.Buffer
	ctx.reset()

	// Generate the code itself
	if typ.static {
		// Iterate through the fields to see if the size can be computed compile
		// time or if runtime resolutions are needed
		var runtime bool
		for i := range typ.opsets {
			if typ.opsets[i].(*opsetStatic).bytes == nil {
				runtime = true
				break
			}
		}
		// If some types require runtime size determination, generate a helper
		// variable to run it on package init
		if runtime {
			fmt.Fprintf(&b, "// Cached static size computed on package init.\n")
			fmt.Fprintf(&b, "var staticSizeCache%s = ", typ.named.Obj().Name())
			for i := range typ.opsets {
				if bytes := typ.opsets[i].(*opsetStatic).bytes; bytes != nil {
					if len(bytes) == 1 {
						fmt.Fprintf(&b, "%d", bytes[0])
					} else {
						fmt.Fprintf(&b, "%d*%d", bytes[0], bytes[1])
					}
				} else {
					typ := typ.types[i].(*types.Pointer).Elem().(*types.Named)
					pkg := typ.Obj().Pkg()
					if pkg.Path() == ctx.pkg.Path() {
						fmt.Fprintf(&b, "new(%s).SizeSSZ()", typ.Obj().Name())
					} else {
						ctx.addImport(pkg.Path(), "")
						fmt.Fprintf(&b, "new(%s.%s).SizeSSZ()", pkg.Name(), typ.Obj().Name())
					}
				}
				if i < len(typ.opsets)-1 {
					fmt.Fprint(&b, " + ")
				}
			}
			fmt.Fprintf(&b, "\n\n// SizeSSZ returns the total size of the static ssz object.\n")
			fmt.Fprintf(&b, "func (obj *%s) SizeSSZ() uint32 {\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "	return staticSizeCache%s\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "}\n")
		} else {
			fmt.Fprint(&b, "// SizeSSZ returns the total size of the static ssz object.\n")
			fmt.Fprintf(&b, "func (obj *%s) SizeSSZ() uint32 {\n", typ.named.Obj().Name())
			fmt.Fprint(&b, "	return ")
			for i := range typ.opsets {
				bytes := typ.opsets[i].(*opsetStatic).bytes
				if len(bytes) == 1 {
					fmt.Fprintf(&b, "%d", bytes[0])
				} else {
					fmt.Fprintf(&b, "%d*%d", bytes[0], bytes[1])
				}
				if i < len(typ.opsets)-1 {
					fmt.Fprint(&b, " + ")
				}
			}
			fmt.Fprintf(&b, "\n}\n")
		}
	} else {
		// Iterate through the fields to see if the static size can be computed
		// compile time or if runtime resolutions are needed even for statics.
		var runtime bool
		for i := range typ.opsets {
			if typ, ok := typ.opsets[i].(*opsetStatic); ok {
				if typ.bytes == nil {
					runtime = true
					break
				}
			}
		}
		// If some types require runtime size determination, generate a helper
		// variable to run it on package init
		if runtime {
			fmt.Fprintf(&b, "// Cached static size computed on package init.\n")
			fmt.Fprintf(&b, "var staticSizeCache%s = ", typ.named.Obj().Name())
			for i := range typ.opsets {
				switch t := typ.opsets[i].(type) {
				case *opsetStatic:
					if t.bytes != nil {
						if len(t.bytes) == 1 {
							fmt.Fprintf(&b, "%d", t.bytes[0])
						} else {
							fmt.Fprintf(&b, "%d*%d", t.bytes[0], t.bytes[1])
						}
					} else {
						typ := typ.types[i].(*types.Pointer).Elem().(*types.Named)
						pkg := typ.Obj().Pkg()
						if pkg.Path() == ctx.pkg.Path() {
							fmt.Fprintf(&b, "new(%s).SizeSSZ()", typ.Obj().Name())
						} else {
							ctx.addImport(pkg.Path(), "")
							fmt.Fprintf(&b, "new(%s.%s).SizeSSZ()", pkg.Name(), typ.Obj().Name())
						}
					}
				case *opsetDynamic:
					fmt.Fprintf(&b, "%d", offsetBytes)
				}
				if i < len(typ.opsets)-1 {
					fmt.Fprint(&b, " + ")
				}
			}
			fmt.Fprintf(&b, "\n\n// SizeSSZ returns either the static size of the object if fixed == true, or\n// the total size otherwise.\n")
			fmt.Fprintf(&b, "func (obj *%s) SizeSSZ(fixed bool) uint32 {\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "	var size = uint32(staticSizeCache%s)\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "	if (fixed) {\n")
			fmt.Fprintf(&b, "		return size\n")
			fmt.Fprintf(&b, "	}\n")
			for i := range typ.opsets {
				if opset, ok := typ.opsets[i].(*opsetDynamic); ok {
					call := generateCall(opset.size, "", "obj."+typ.fields[i])
					fmt.Fprintf(&b, "	size += ssz.%s\n", call)
				}
			}
			fmt.Fprintf(&b, "\n")
			fmt.Fprintf(&b, "	return size\n")
			fmt.Fprintf(&b, "}\n")
		} else {
			fmt.Fprintf(&b, "\n\n// SizeSSZ returns either the static size of the object if fixed == true, or\n// the total size otherwise.\n")
			fmt.Fprintf(&b, "func (obj *%s) SizeSSZ(fixed bool) uint32 {\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "	var size = uint32(")
			for i := range typ.opsets {
				switch t := typ.opsets[i].(type) {
				case *opsetStatic:
					if len(t.bytes) == 1 {
						fmt.Fprintf(&b, "%d", t.bytes[0])
					} else {
						fmt.Fprintf(&b, "%d*%d", t.bytes[0], t.bytes[1])
					}
				case *opsetDynamic:
					fmt.Fprintf(&b, "%d", offsetBytes)
				}
				if i < len(typ.opsets)-1 {
					fmt.Fprint(&b, " + ")
				}
			}
			fmt.Fprintf(&b, ")\n")
			fmt.Fprintf(&b, "	if (fixed) {\n")
			fmt.Fprintf(&b, "		return size\n")
			fmt.Fprintf(&b, "	}\n")
			for i := range typ.opsets {
				if opset, ok := typ.opsets[i].(*opsetDynamic); ok {
					call := generateCall(opset.size, "", "obj."+typ.fields[i])
					fmt.Fprintf(&b, "	size += ssz.%s\n", call)
				}
			}
			fmt.Fprintf(&b, "\n")
			fmt.Fprintf(&b, "	return size\n")
			fmt.Fprintf(&b, "}\n")
		}
	}
	return b.Bytes(), nil
}

func generateDefineSSZ(ctx *genContext, typ *sszContainer) ([]byte, error) {
	var b bytes.Buffer
	ctx.reset()

	// Add a needed import of the ssz encoder
	ctx.addImport(sszPkgPath, "")

	// Iterate through the fields names to compute some comment formatting mods
	var (
		maxFieldLength = 0
		maxBytes       = 1
	)
	for i, field := range typ.fields {
		maxFieldLength = max(maxFieldLength, len(field))
		switch opset := typ.opsets[i].(type) {
		case *opsetStatic:
			switch len(opset.bytes) {
			case 1:
				maxBytes = max(maxBytes, opset.bytes[0])
			case 2:
				maxBytes = max(maxBytes, opset.bytes[0]*opset.bytes[1])
			}
		case *opsetDynamic:
			maxBytes = max(maxBytes, offsetBytes) // offset size
		}
	}
	var (
		indexRule = fmt.Sprintf("%%%dd", int(math.Ceil(math.Log10(float64(len(typ.fields))))))
		nameRule  = fmt.Sprintf("%%%ds", maxFieldLength)
		sizeRule  = fmt.Sprintf("%d", int(math.Ceil(math.Log10(float64(maxBytes)))))
	)
	// Generate the code itself
	fmt.Fprint(&b, "// DefineSSZ defines how an object is encoded/decoded.\n")
	fmt.Fprintf(&b, "func (obj *%s) DefineSSZ(codec *ssz.Codec) {\n", typ.named.Obj().Name())
	if !typ.static {
		fmt.Fprint(&b, "	// Define the static data (fields and dynamic offsets)\n")
	}
	for i := 0; i < len(typ.fields); i++ {
		field := typ.fields[i]
		switch opset := typ.opsets[i].(type) {
		case *opsetStatic:
			call := generateCall(opset.define, "codec", "obj."+field, opset.bytes...)
			switch len(opset.bytes) {
			case 0:
				typ := typ.types[i].(*types.Pointer).Elem().(*types.Named)
				fmt.Fprintf(&b, "	ssz.%s // Field  ("+indexRule+") - "+nameRule+" - %"+sizeRule+"s bytes (%s)\n", call, i, field, "?", typ.Obj().Name())
			case 1:
				fmt.Fprintf(&b, "	ssz.%s // Field  ("+indexRule+") - "+nameRule+" - %"+sizeRule+"d bytes\n", call, i, field, opset.bytes[0])
			case 2:
				fmt.Fprintf(&b, "	ssz.%s // Field  ("+indexRule+") - "+nameRule+" - %"+sizeRule+"d bytes\n", call, i, field, opset.bytes[0]*opset.bytes[1])
			}
		case *opsetDynamic:
			call := generateCall(opset.defineOffset, "codec", "obj."+field, opset.limits...)
			fmt.Fprintf(&b, "	ssz.%s // Offset ("+indexRule+") - "+nameRule+" - %"+sizeRule+"d bytes\n", call, i, field, offsetBytes)
		}
	}
	if !typ.static {
		fmt.Fprint(&b, "\n	// Define the dynamic data (fields)\n")
		for i := 0; i < len(typ.fields); i++ {
			field := typ.fields[i]
			if opset, ok := (typ.opsets[i]).(*opsetDynamic); ok {
				call := generateCall(opset.defineContent, "codec", "obj."+field, opset.limits...)
				fmt.Fprintf(&b, "	ssz.%s // Field  ("+indexRule+") - "+nameRule+" - ? bytes\n", call, i, field)
			}
		}
	}
	fmt.Fprint(&b, "}\n")
	return b.Bytes(), nil
}

// generateCall parses a Go template and fills it with the provided data. This
// could be done more optimally, but we really don't care for a code generator.
func generateCall(tmpl string, recv string, field string, limits ...int) string {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		panic(err)
	}
	d := map[string]interface{}{
		"Codec": recv,
		"Field": field,
	}
	if len(limits) > 0 {
		d["MaxSize"] = limits[len(limits)-1]
	}
	if len(limits) > 1 {
		d["MaxItems"] = limits[len(limits)-2]
	}
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, d); err != nil {
		panic(err)
	}
	return string(buf.Bytes())
}
