// Copyright (c) 2026 Kirill Scherba <kirill@scherba.ru>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package parser parses Go source files and extracts function signatures via AST.
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// TypeKind classifies a Go type expression.
type TypeKind int

const (
	KindInvalid TypeKind = iota
	KindBool
	KindInt
	KindFloat
	KindString
	KindSlice
	KindArray
	KindMap
	KindChan
	KindFunc
	KindStruct
	KindInterface
	KindPointer
	KindNamed
	KindGeneric
)

// TypeInfo describes a Go type expression found in source.
type TypeInfo struct {
	Name      string   // Type name as written in source (e.g. "string", "[]int", "MyType")
	Kind      TypeKind // Classification
	IsPointer bool
	IsSlice   bool
	IsMap     bool
	IsChan    bool
	IsArray   bool
	Elem      *TypeInfo // Element type for containers / pointer target
	Key       *TypeInfo // Key type for maps
	PkgPath   string    // Import path for imported types (e.g. "net/http")
	Expr      ast.Expr  // Original AST expression (for advanced analysis)
}

// ParamInfo describes a function parameter or result.
type ParamInfo struct {
	Name string   // Parameter name (may be empty)
	Type TypeInfo // Parameter type
}

// FuncInfo describes a Go function or method declaration.
type FuncInfo struct {
	Name       string      // Function name
	Receiver   *ParamInfo  // nil for plain functions
	Params     []ParamInfo // Input parameters
	Returns    []TypeInfo  // Return types (empty for void)
	IsExported bool
	IsMethod   bool
	IsGeneric  bool
	TypeParams []TypeInfo // Generic type parameters
	IsVariadic bool       // Has ...T final parameter
	Pos        token.Pos  // Source position
	File       string     // Source file path
}

// PackageName extracts the package name from a Go source file.
func PackageName(path string) (string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return "", err
	}
	if f.Name == nil {
		return "", fmt.Errorf("no package declaration in %s", path)
	}
	return f.Name.Name, nil
}

// ParseFile parses a Go source file and returns all function declarations found.
func ParseFile(path string) ([]FuncInfo, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var funcs []FuncInfo
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			info := extractFuncInfo(fn, path)
			funcs = append(funcs, info)
		}
	}
	return funcs, nil
}

// FindFunc finds a function by name in a Go source file.
func FindFunc(path string, name string) (*FuncInfo, error) {
	funcs, err := ParseFile(path)
	if err != nil {
		return nil, err
	}
	for i := range funcs {
		if funcs[i].Name == name {
			return &funcs[i], nil
		}
	}
	return nil, nil
}

func extractFuncInfo(fn *ast.FuncDecl, path string) FuncInfo {
	info := FuncInfo{
		Name:       fn.Name.Name,
		IsExported: ast.IsExported(fn.Name.Name),
		IsMethod:   fn.Recv != nil,
		Pos:        fn.Pos(),
		File:       path,
	}

	// Receiver
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		info.Receiver = extractParam(fn.Recv.List[0])
	}

	// Type parameters (generics)
	if fn.Type != nil && fn.Type.TypeParams != nil {
		info.IsGeneric = true
		for _, field := range fn.Type.TypeParams.List {
			for _, ident := range field.Names {
				info.TypeParams = append(info.TypeParams, TypeInfo{
					Name: ident.Name,
					Kind: KindGeneric,
					Expr: field.Type,
				})
			}
		}
	}

	// Parameters
	if fn.Type != nil && fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			// Variadic detection: last param has *ast.Ellipsis
			if _, ok := field.Type.(*ast.Ellipsis); ok {
				info.IsVariadic = true
			}
			for _, name := range field.Names {
				info.Params = append(info.Params, ParamInfo{
					Name: name.Name,
					Type: extractType(field.Type),
				})
			}
			// No names: unnamed parameter
			if len(field.Names) == 0 {
				info.Params = append(info.Params, ParamInfo{
					Name: "",
					Type: extractType(field.Type),
				})
			}
		}
	}

	// Return values
	if fn.Type != nil && fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			// Named return values: extract name if present
			if len(field.Names) > 0 {
				for _, name := range field.Names {
					_ = name // We store only TypeInfo in Returns for simplicity
				}
			}
			info.Returns = append(info.Returns, extractType(field.Type))
		}
	}

	return info
}

func extractParam(field *ast.Field) *ParamInfo {
	param := ParamInfo{Type: extractType(field.Type)}
	if len(field.Names) > 0 {
		param.Name = field.Names[0].Name
	}
	return &param
}

func extractType(expr ast.Expr) TypeInfo {
	name := typeToString(expr)
	kind, isPtr, isSlice, isMap, isChan, isArray, elem, key := classifyType(expr)

	return TypeInfo{
		Name:      name,
		Kind:      kind,
		IsPointer: isPtr,
		IsSlice:   isSlice,
		IsMap:     isMap,
		IsChan:    isChan,
		IsArray:   isArray,
		Elem:      elem,
		Key:       key,
		Expr:      expr,
	}
}

// typeToString returns a readable string representation of a type expression.
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeToString(t.Elt)
		}
		return "[" + typeToString(t.Len) + "]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.ChanType:
		return "chan " + typeToString(t.Value)
	case *ast.FuncType:
		return "func(...)" // Simplified
	case *ast.StructType:
		return "struct{...}"
	case *ast.InterfaceType:
		return "interface{...}"
	case *ast.Ellipsis:
		return "..." + typeToString(t.Elt)
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

// classifyType determines TypeKind and container flags from an ast.Expr.
func classifyType(expr ast.Expr) (kind TypeKind, isPtr, isSlice, isMap, isChan, isArray bool, elem, key *TypeInfo) {
	switch t := expr.(type) {
	case *ast.Ident:
		kind = kindFromName(t.Name)
	case *ast.StarExpr:
		isPtr = true
		kind = KindPointer
		e := extractType(t.X)
		elem = &e
	case *ast.ArrayType:
		if t.Len == nil {
			isSlice = true
			kind = KindSlice
		} else {
			isArray = true
			kind = KindArray
		}
		e := extractType(t.Elt)
		elem = &e
	case *ast.MapType:
		isMap = true
		kind = KindMap
		k := extractType(t.Key)
		key = &k
		e := extractType(t.Value)
		elem = &e
	case *ast.ChanType:
		isChan = true
		kind = KindChan
		e := extractType(t.Value)
		elem = &e
	case *ast.FuncType:
		kind = KindFunc
	case *ast.StructType:
		kind = KindStruct
	case *ast.InterfaceType:
		kind = KindInterface
	case *ast.Ellipsis:
		isSlice = true
		kind = KindSlice // Variadic param is effectively a slice
		e := extractType(t.Elt)
		elem = &e
	case *ast.SelectorExpr:
		// Imported type: e.g. http.Handler, pkg.MyType
		kind = KindNamed
		if ident, ok := t.X.(*ast.Ident); ok {
			// We can't resolve the import path without go/types,
			// so we just store the package name as part of Name
			_ = ident.Name
		}
	default:
		kind = KindInvalid
	}
	return
}

func kindFromName(name string) TypeKind {
	switch name {
	case "bool":
		return KindBool
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"uintptr", "byte", "rune":
		return KindInt
	case "float32", "float64":
		return KindFloat
	case "string":
		return KindString
	case "any", "interface":
		return KindInterface
	default:
		if strings.HasPrefix(name, "T") || strings.HasPrefix(name, "U") ||
			strings.HasPrefix(name, "V") || strings.HasPrefix(name, "K") || strings.HasPrefix(name, "E") {
			// Heuristic for generic type parameters
			return KindGeneric
		}
		return KindNamed
	}
}
