// Copyright (c) 2026 Kirill Scherba <kirill@scherba.ru>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package fuzz generates fuzz test scaffolding.
package fuzz

import (
	"fmt"
	"strings"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
	"github.com/kirill-scherba/go-test-oracle/internal/parser"
	"github.com/kirill-scherba/go-test-oracle/internal/score"
)

// New creates a fuzz template generator.
func New() generator.Generator { return &fuzzGen{} }

type fuzzGen struct{}

func (f *fuzzGen) Name() string { return "fuzz" }

// seedValue returns a type-aware seed literal for the fuzz corpus.
func seedValue(t parser.TypeInfo) string {
	switch t.Kind {
	case parser.KindInt:
		return "int64(0), int64(-1), int64(42)"
	case parser.KindFloat:
		return "float64(0.0), float64(-1.0), float64(3.14)"
	case parser.KindString:
		return `"", "hello", "\x00"`
	case parser.KindBool:
		return "true, false, true"
	case parser.KindSlice:
		return `[]byte{}, []byte("abc"), []byte{0, 255}`
	default:
		// Fallback: use string as catch-all
		return `"", "hello", "world"`
	}
}

// fuzzArgType returns the argument type used in f.Fuzz(func(t *testing.T, ...)).
func fuzzArgType(t parser.TypeInfo) string {
	switch t.Kind {
	case parser.KindInt:
		return "int64"
	case parser.KindFloat:
		return "float64"
	case parser.KindString:
		return "string"
	case parser.KindBool:
		return "bool"
	case parser.KindSlice:
		// Fuzzing a slice requires concrete type — default to []byte
		return "[]byte"
	default:
		return "string"
	}
}

// fuzzConversion returns the expression to convert from fuzz arg to the target type.
func fuzzConversion(t parser.TypeInfo, varName string) string {
	switch t.Kind {
	case parser.KindInt:
		return fmt.Sprintf("int(%s)", varName)
	case parser.KindFloat:
		return fmt.Sprintf("%s", varName) // already float64
	case parser.KindString:
		return varName
	case parser.KindBool:
		return varName
	case parser.KindSlice:
		return varName // already []byte
	default:
		return varName
	}
}

func (f *fuzzGen) Generate(fn *parser.FuncInfo) (*generator.Result, error) {
	if len(fn.Params) == 0 {
		return f.generateNoParams(fn), nil
	}

	s := score.Calculate(fn)

	// Determine seed corpus values
	seeds := []string{}
	argTypes := []string{}
	conversions := []string{}
	argNames := []string{}

	for i, p := range fn.Params {
		argName := fmt.Sprintf("arg%d", i)
		if p.Name != "" && p.Name != "_" {
			argName = p.Name
		}
		argNames = append(argNames, argName)
		argTypes = append(argTypes, fuzzArgType(p.Type))
		conversions = append(conversions, fuzzConversion(p.Type, argName))
		seeds = append(seeds, seedValue(p.Type))
	}

	// Build f.Add calls (cross-product of first seed for each param)
	// Simplified: single seed combination
	seedVals := make([]string, len(seeds))
	for i, s := range seeds {
		parts := strings.Split(s, ", ")
		if len(parts) > 0 {
			seedVals[i] = strings.TrimSpace(parts[0])
		}
	}
	addLine := fmt.Sprintf("\tf.Add(%s)", strings.Join(seedVals, ", "))

	// Build fuzz function signature
	fuzzArgs := make([]string, len(argTypes))
	for i, at := range argTypes {
		fuzzArgs[i] = fmt.Sprintf("%s %s", argNames[i], at)
	}

	// Build call expression
	callExpr := fmt.Sprintf("%s(%s)", fn.Name, strings.Join(conversions, ", "))
	// Handle methods: prepend receiver
	if fn.IsMethod && fn.Receiver != nil {
		recType := fn.Receiver.Type.Name
		if fn.Receiver.Type.IsPointer {
			recType = strings.TrimPrefix(recType, "*")
		}
		callExpr = fmt.Sprintf("%s{}.%s(%s)", recType, fn.Name, strings.Join(conversions, ", "))
	}

	skipLine := ""
	if s.Skip {
		skipLine = "\t\tt.Skip(\"TODO: define expected behavior\")\n"
	}

	code := fmt.Sprintf(`func Fuzz%s(f *testing.F) {
%s
	f.Fuzz(func(t *testing.T, %s) {
		_ = %s
%s	})
}`, fn.Name, addLine, strings.Join(fuzzArgs, ", "), callExpr, skipLine)

	return &generator.Result{
		Code:       code,
		Confidence: s.Confidence,
		Reason:     s.Reason + " (fuzz)",
		Imports:    nil,
	}, nil
}

func (f *fuzzGen) generateNoParams(fn *parser.FuncInfo) *generator.Result {
	s := score.Calculate(fn)

	var callExpr string
	if fn.IsMethod && fn.Receiver != nil {
		recType := fn.Receiver.Type.Name
		if fn.Receiver.Type.IsPointer {
			recType = strings.TrimPrefix(recType, "*")
		}
		callExpr = fmt.Sprintf("%s{}.%s()", recType, fn.Name)
	} else {
		callExpr = fmt.Sprintf("%s()", fn.Name)
	}

	skipLine := ""
	if s.Skip {
		skipLine = "\tt.Skip(\"TODO: define expected behavior\")\n"
	}

	code := fmt.Sprintf(`func Fuzz%s(f *testing.F) {
	f.Fuzz(func(t *testing.T) {
		_ = %s
%s	})
}`, fn.Name, callExpr, skipLine)

	return &generator.Result{
		Code:       code,
		Confidence: s.Confidence,
		Reason:     s.Reason + " (fuzz no-params)",
		Imports:    nil,
	}
}
