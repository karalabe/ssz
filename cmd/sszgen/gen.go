// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"fmt"
	"go/types"
	"html/template"
	"io"
	"math"
	"sort"
	"strings"
)

const (
	offsetBytes = 4
	sszPkgPath  = "github.com/karalabe/ssz"
)

type genContext struct {
	pkg     *types.Package
	imports map[string]string
}

func newGenContext(pkg *types.Package) *genContext {
	return &genContext{
		pkg:     pkg,
		imports: make(map[string]string),
	}
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
	//fmt.Fprintf(&b, ")\n")
	return b.Bytes()
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
	//fmt.Println(string(bytes.Join(codes, []byte("\n"))))
	return bytes.Join(codes, []byte("\n")), nil
}

// generateStaticSizeAccumulator is a helper to iterate over all the fields and
// accumulate the static sizes into a `size` variable based on fork constraints.
func generateStaticSizeAccumulator(w io.Writer, ctx *genContext, typ *sszContainer) {
	for i := range typ.opsets {
		switch {
		case typ.forks[i] == "" && i == 0:
			fmt.Fprintf(w, "	size = ")
		case typ.forks[i] == "" && typ.forks[i-1] == "":
			fmt.Fprintf(w, " + ")
		case typ.forks[i] == "" && typ.forks[i-1] != "":
			fmt.Fprintf(w, "\n	size += ")
		case typ.forks[i] != "" && i > 0 && typ.forks[i-1] != typ.forks[i]:
			fmt.Fprintf(w, "\n")
		}
		if typ.forks[i] != "" {
			if i == 0 || typ.forks[i] != typ.forks[i-1] {
				if typ.forks[i][0] == '!' {
					fmt.Fprintf(w, "	if sizer.Fork() < ssz.Fork%s {\n", typ.forks[i][1:])
				} else {
					fmt.Fprintf(w, "	if sizer.Fork() >= ssz.Fork%s {\n", typ.forks[i])
				}
				fmt.Fprintf(w, "		size += ")
			} else {
				fmt.Fprintf(w, " + ")
			}
		}
		switch t := typ.opsets[i].(type) {
		case *opsetStatic:
			if t.bytes != nil {
				if len(t.bytes) == 1 {
					fmt.Fprintf(w, "%d", t.bytes[0])
				} else {
					fmt.Fprintf(w, "%d*%d", t.bytes[0], t.bytes[1])
				}
			} else {
				typ := typ.types[i].(*types.Pointer).Elem().(*types.Named)
				pkg := typ.Obj().Pkg()
				if pkg.Path() == ctx.pkg.Path() {
					fmt.Fprintf(w, "(*%s)(nil).SizeSSZ(sizer)", typ.Obj().Name())
				} else {
					ctx.addImport(pkg.Path(), "")
					fmt.Fprintf(w, "(*%s.%s)(nil).SizeSSZ(sizer)", pkg.Name(), typ.Obj().Name())
				}
			}
		case *opsetDynamic:
			fmt.Fprintf(w, "%d", offsetBytes)
		}
		if typ.forks[i] != "" && (i == len(typ.forks)-1 || typ.forks[i] != typ.forks[i+1]) {
			fmt.Fprintf(w, "\n	}")
		}
	}
	fmt.Fprintf(w, "	\n")
}

func generateSizeSSZ(ctx *genContext, typ *sszContainer) ([]byte, error) {
	var b bytes.Buffer

	// Generate the code itself
	if typ.static {
		// Iterate through the fields to see if the size can be computed compile
		// time or if runtime resolutions are needed.
		var (
			runtime  bool
			monolith bool
		)
		for i := range typ.opsets {
			if typ.opsets[i].(*opsetStatic).bytes == nil {
				runtime = true
			}
			if typ.forks[i] != "" {
				monolith = true
			}
		}
		// If some types require runtime size determination, generate a helper
		// variable to run it on package init
		if runtime {
			fmt.Fprintf(&b, "// Cached static size computed on package init.\n")
			fmt.Fprintf(&b, "var staticSizeCache%s = ssz.PrecomputeStaticSizeCache((*%s)(nil))\n\n", typ.named.Obj().Name(), typ.named.Obj().Name())

			fmt.Fprintf(&b, "// SizeSSZ returns the total size of the static ssz object.\n")
			fmt.Fprintf(&b, "func (obj *%s) SizeSSZ(sizer *ssz.Sizer) (size uint32) {\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "	if fork := int(sizer.Fork()); fork < len(staticSizeCache%s) {\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "		return staticSizeCache%s[fork]\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "	}\n")

			generateStaticSizeAccumulator(&b, ctx, typ)
			fmt.Fprintf(&b, "	return size\n}\n")
		} else {
			fmt.Fprint(&b, "// SizeSSZ returns the total size of the static ssz object.\n")
			if monolith {
				fmt.Fprintf(&b, "func (obj *%s) SizeSSZ(sizer *ssz.Sizer) (size uint32) {\n", typ.named.Obj().Name())
				generateStaticSizeAccumulator(&b, ctx, typ)
				fmt.Fprintf(&b, "	return size\n}\n")
			} else {
				fmt.Fprintf(&b, "func (obj *%s) SizeSSZ(sizer *ssz.Sizer) uint32 {\n", typ.named.Obj().Name())
				fmt.Fprintf(&b, "	return ")
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
		}
	} else {
		// Iterate through the fields to see if the static size can be computed
		// compile time or if runtime resolutions are needed even for statics.
		var runtime bool
		for i := range typ.opsets {
			if typ, ok := typ.opsets[i].(*opsetStatic); ok {
				if typ.bytes == nil {
					runtime = true
				}
			}
		}
		// If some types require runtime size determination, generate a helper
		// variable to run it on package init
		if runtime {
			fmt.Fprintf(&b, "// Cached static size computed on package init.\n")
			fmt.Fprintf(&b, "var staticSizeCache%s = ssz.PrecomputeStaticSizeCache((*%s)(nil))\n\n", typ.named.Obj().Name(), typ.named.Obj().Name())

			fmt.Fprintf(&b, "// SizeSSZ returns either the static size of the object if fixed == true, or\n// the total size otherwise.\n")
			fmt.Fprintf(&b, "func (obj *%s) SizeSSZ(sizer *ssz.Sizer, fixed bool) (size uint32) {\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "	// Load static size if already precomputed, calculate otherwise\n")
			fmt.Fprintf(&b, "	if fork := int(sizer.Fork()); fork < len(staticSizeCache%s) {\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "		size = staticSizeCache%s[fork]\n", typ.named.Obj().Name())
			fmt.Fprintf(&b, "	} else {\n")
			generateStaticSizeAccumulator(&b, ctx, typ)
			fmt.Fprintf(&b, "	}\n")
			fmt.Fprintf(&b, "	// Either return the static size or accumulate the dynamic too\n")
			fmt.Fprintf(&b, "	if (fixed) {\n")
			fmt.Fprintf(&b, "		return size\n")
			fmt.Fprintf(&b, "	}\n")
			var (
				dynFields []string
				dynOpsets []opset
				dynForks  []string
			)
			for i := 0; i < len(typ.fields); i++ {
				if _, ok := (typ.opsets[i]).(*opsetDynamic); ok {
					dynFields = append(dynFields, typ.fields[i])
					dynOpsets = append(dynOpsets, typ.opsets[i])
					dynForks = append(dynForks, typ.forks[i])
				}
			}
			for i := range dynFields {
				if dynForks[i] != "" && (i == 0 || dynForks[i] != dynForks[i-1]) {
					if dynForks[i][0] == '!' {
						fmt.Fprintf(&b, "	if sizer.Fork() < ssz.Fork%s {\n", dynForks[i][1:])
					} else {
						fmt.Fprintf(&b, "	if sizer.Fork() >= ssz.Fork%s {\n", dynForks[i])
					}
				}
				call := generateCall(dynOpsets[i].(*opsetDynamic).size, "", "sizer", "obj."+dynFields[i])
				fmt.Fprintf(&b, "	size += ssz.%s\n", call)
				if dynForks[i] != "" && (i == len(dynForks)-1 || dynForks[i] != dynForks[i+1]) {
					fmt.Fprintf(&b, "	}\n")
				}
			}
			if dynForks[len(dynForks)-1] == "" {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "	return size\n")
			fmt.Fprintf(&b, "}\n")
		} else {
			fmt.Fprintf(&b, "\n\n// SizeSSZ returns either the static size of the object if fixed == true, or\n// the total size otherwise.\n")
			fmt.Fprintf(&b, "func (obj *%s) SizeSSZ(sizer *ssz.Sizer, fixed bool) (size uint32) {\n", typ.named.Obj().Name())
			generateStaticSizeAccumulator(&b, ctx, typ)
			fmt.Fprintf(&b, "	if (fixed) {\n")
			fmt.Fprintf(&b, "		return size\n")
			fmt.Fprintf(&b, "	}\n")

			var (
				dynFields []string
				dynOpsets []opset
				dynForks  []string
			)
			for i := 0; i < len(typ.fields); i++ {
				if _, ok := (typ.opsets[i]).(*opsetDynamic); ok {
					dynFields = append(dynFields, typ.fields[i])
					dynOpsets = append(dynOpsets, typ.opsets[i])
					dynForks = append(dynForks, typ.forks[i])
				}
			}
			for i := range dynFields {
				if dynForks[i] != "" && (i == 0 || dynForks[i] != dynForks[i-1]) {
					if dynForks[i][0] == '!' {
						fmt.Fprintf(&b, "	if sizer.Fork() < ssz.Fork%s {\n", dynForks[i][1:])
					} else {
						fmt.Fprintf(&b, "	if sizer.Fork() >= ssz.Fork%s {\n", dynForks[i])
					}
				}
				call := generateCall(dynOpsets[i].(*opsetDynamic).size, "", "sizer", "obj."+dynFields[i])
				fmt.Fprintf(&b, "	size += ssz.%s\n", call)
				if dynForks[i] != "" && (i == len(dynForks)-1 || dynForks[i] != dynForks[i+1]) {
					fmt.Fprintf(&b, "	}\n")
				}
			}
			if dynForks[len(dynForks)-1] == "" {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "	return size\n")
			fmt.Fprintf(&b, "}\n")
		}
	}
	return b.Bytes(), nil
}

func generateDefineSSZ(ctx *genContext, typ *sszContainer) ([]byte, error) {
	var b bytes.Buffer

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
			call := generateCall(opset.define, typ.forks[i], "codec", "obj."+field, opset.bytes...)
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
			call := generateCall(opset.defineOffset, typ.forks[i], "codec", "obj."+field, opset.limits...)
			fmt.Fprintf(&b, "	ssz.%s // Offset ("+indexRule+") - "+nameRule+" - %"+sizeRule+"d bytes\n", call, i, field, offsetBytes)
		}
	}
	if !typ.static {
		fmt.Fprint(&b, "\n	// Define the dynamic data (fields)\n")
		var (
			dynIndices []int
			dynFields  []string
			dynOpsets  []opset
			dynForks   []string
		)
		for i := 0; i < len(typ.fields); i++ {
			if _, ok := (typ.opsets[i]).(*opsetDynamic); ok {
				dynIndices = append(dynIndices, i)
				dynFields = append(dynFields, typ.fields[i])
				dynOpsets = append(dynOpsets, typ.opsets[i])
				dynForks = append(dynForks, typ.forks[i])
			}
		}
		for i := 0; i < len(dynFields); i++ {
			opset := (dynOpsets[i]).(*opsetDynamic)

			call := generateCall(opset.defineContent, dynForks[i], "codec", "obj."+dynFields[i], opset.limits...)
			fmt.Fprintf(&b, "	ssz.%s // Field  ("+indexRule+") - "+nameRule+" - ? bytes\n", call, dynIndices[i], dynFields[i])
		}
	}
	fmt.Fprint(&b, "}\n")
	return b.Bytes(), nil
}

// generateCall parses a Go template and fills it with the provided data. This
// could be done more optimally, but we really don't care for a code generator.
func generateCall(tmpl string, fork string, recv string, field string, limits ...int) string {
	// Generate the base call without taking forks into consideration
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		panic(err)
	}
	d := map[string]interface{}{
		"Sizer": recv,
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
	call := string(buf.Bytes())

	// If a fork filter was specified, inject it now into the call
	if fork != "" {
		// Mutate the call to the fork variant
		call = strings.ReplaceAll(call, "(", "OnFork(")

		// Inject a fork filter as the last parameter
		var filter string
		if fork[0] == '!' {
			filter = fmt.Sprintf("ssz.ForkFilter{Removed: ssz.Fork%s}", fork[1:])
		} else {
			filter = fmt.Sprintf("ssz.ForkFilter{Added: ssz.Fork%s}", fork)
		}
		call = strings.ReplaceAll(call, ")", ","+filter+")")
	}
	return call
}
