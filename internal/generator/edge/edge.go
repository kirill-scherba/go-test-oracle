// Copyright (c) 2026 Kirill Scherba <kirill@scherba.ru>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package edge generates edge-case test scaffolding.
package edge

import (
	"fmt"
	"strings"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

// New creates an edge-case generator.
func New() generator.Generator { return &edgeGen{} }

type edgeGen struct{}

func (e *edgeGen) Name() string { return "edge" }

// EdgeCase describes one edge value for a parameter.
type edgeCase struct {
	Name        string // e.g. "nil", "zero", "empty"
	Code        string // Go literal or expression
	NeedsMath   bool   // Requires "math" import
	NeedsString bool   // Requires "strings" import
}

// edgeMap maps TypeKind to known edge values.
var edgeMap = map[parser.TypeKind][]edgeCase{
	parser.KindInt: {
		{Name: "zero", Code: "0"},
		{Name: "negative", Code: "-1"},
		{Name: "positive", Code: "1"},
		{Name: "max", Code: "math.MaxInt64", NeedsMath: true},
		{Name: "min", Code: "math.MinInt64", NeedsMath: true},
	},
	parser.KindFloat: {
		{Name: "zero", Code: "0.0"},
		{Name: "negative", Code: "-1.0"},
		{Name: "positive", Code: "1.0"},
		{Name: "nan", Code: "math.NaN()", NeedsMath: true},
		{Name: "inf", Code: "math.Inf(1)", NeedsMath: true},
		{Name: "neg_inf", Code: "math.Inf(-1)", NeedsMath: true},
	},
	parser.KindString: {
		{Name: "empty", Code: `""`},
		{Name: "space", Code: `" "`},
		{Name: "unicode", Code: `"Привет"`},
		{Name: "null_byte", Code: `"\x00"`},
		{Name: "long", Code: `strings.Repeat("a", 1000)`, NeedsString: true},
	},
	parser.KindBool: {
		{Name: "true", Code: "true"},
		{Name: "false", Code: "false"},
	},
	parser.KindSlice: {
		{Name: "nil", Code: "nil"},
		{Name: "empty", Code: "nil /* empty slice: []T{} requires concrete type */"},
	},
	parser.KindMap: {
		{Name: "nil", Code: "nil"},
		{Name: "empty", Code: "nil /* empty map: map[K]V{} requires concrete type */"},
	},
	parser.KindPointer: {
		{Name: "nil", Code: "nil"},
	},
	parser.KindChan: {
		{Name: "nil", Code: "nil"},
	},
	parser.KindInterface: {
		{Name: "nil", Code: "nil"},
		{Name: "string", Code: `"hello"`},
		{Name: "int", Code: "42"},
	},
	parser.KindFunc: {
		{Name: "nil", Code: "nil"},
	},
}

// typicalValue returns a sensible default value for a type when it is NOT the edge under test.
func typicalValue(t parser.TypeInfo) string {
	switch t.Kind {
	case parser.KindInt:
		return "42"
	case parser.KindFloat:
		return "1.5"
	case parser.KindString:
		return `"hello"`
	case parser.KindBool:
		return "true"
	case parser.KindSlice:
		return "nil /* typical: []T{1,2,3} */"
	case parser.KindMap:
		return "nil /* typical: map[K]V{...} */"
	case parser.KindPointer:
		return "nil"
	case parser.KindChan:
		return "nil"
	case parser.KindInterface:
		return `"hello"`
	case parser.KindFunc:
		return "nil"
	case parser.KindStruct:
		return fmt.Sprintf("%s{}", t.Name)
	case parser.KindNamed:
		return fmt.Sprintf("%s{}", t.Name) // zero value of named type
	case parser.KindGeneric:
		return "nil /* generic type — use zero value */"
	default:
		return fmt.Sprintf("%s{}", t.Name)
	}
}

func (e *edgeGen) Generate(fn *parser.FuncInfo) (*generator.Result, error) {
	if len(fn.Params) == 0 {
		// No parameters — generate a simple invocation test
		return e.generateNoParams(fn), nil
	}

	// Build field list for test struct
	var fields []string
	var imports []string
	hasMath := false
	hasStrings := false

	// Name of the test description field
	testNameField := e.uniquify("name", fn.Params)

	fields = append(fields, fmt.Sprintf("%s string", testNameField))

	// Track parameter names (may collide with testNameField)
	paramNames := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		n := p.Name
		if n == "" || n == "_" {
			n = fmt.Sprintf("arg%d", i)
		}
		// Ensure uniqueness with testNameField and other params
		n = e.uniquifyAmong(n, append([]string{testNameField}, paramNames[:i]...))
		paramNames[i] = n
		fields = append(fields, fmt.Sprintf("%s %s", n, p.Type.Name))
	}

	// Build test cases — one edge per parameter
	var cases []string
	for i, p := range fn.Params {
		edges := edgeMap[p.Type.Kind]
		if len(edges) == 0 {
			// Unknown type — single case with zero value
			edges = []edgeCase{{Name: "zero", Code: typicalValue(p.Type)}}
		}
		for _, edge := range edges {
			if edge.NeedsMath {
				hasMath = true
			}
			if edge.NeedsString {
				hasStrings = true
			}
			vals := make([]string, len(fn.Params))
			for j := range fn.Params {
				if j == i {
					vals[j] = edge.Code
				} else {
					vals[j] = typicalValue(fn.Params[j].Type)
				}
			}
			caseName := fmt.Sprintf("%s_%s", paramNames[i], edge.Name)
			lines := []string{
				fmt.Sprintf("\t\t\t%s: %q,", testNameField, caseName),
			}
			for j, v := range vals {
				lines = append(lines, fmt.Sprintf("\t\t\t%s: %s,", paramNames[j], v))
			}
			cases = append(cases, fmt.Sprintf("\t\t{\n%s\n\t\t},", strings.Join(lines, "\n")))
		}
	}

	if hasMath {
		imports = append(imports, "math")
	}
	if hasStrings {
		imports = append(imports, "strings")
	}

	// Determine confidence
	confidence, reason := e.score(fn)

	// Build the function body
	var callExpr string
	if fn.IsMethod {
		recType := fn.Receiver.Type.Name
		if fn.Receiver.Type.IsPointer {
			recType = strings.TrimPrefix(recType, "*")
		}
		callArgs := make([]string, len(paramNames))
		for i, n := range paramNames {
			callArgs[i] = fmt.Sprintf("tt.%s", n)
		}
		callExpr = fmt.Sprintf("%s{}.%s(%s)", recType, fn.Name, strings.Join(callArgs, ", "))
	} else {
		callArgs := make([]string, len(paramNames))
		for i, n := range paramNames {
			callArgs[i] = fmt.Sprintf("tt.%s", n)
		}
		callExpr = fmt.Sprintf("%s(%s)", fn.Name, strings.Join(callArgs, ", "))
	}

	skipLine := ""
	if confidence < 0.6 {
		skipLine = "\t\t\tt.Skip(\"TODO: define expected behavior\")\n"
	}

	code := fmt.Sprintf(`func Test%s_EdgeCases(t *testing.T) {
	tests := []struct {
%s
	}{
%s
	}
	for _, tt := range tests {
		t.Run(tt.%s, func(t *testing.T) {
			_ = %s
%s		})
	}
}`, fn.Name, strings.Join(fields, "\n"), strings.Join(cases, "\n"), testNameField, callExpr, skipLine)

	return &generator.Result{
		Code:       code,
		Confidence: confidence,
		Reason:     reason,
		Imports:    imports,
	}, nil
}

func (e *edgeGen) generateNoParams(fn *parser.FuncInfo) *generator.Result {
	confidence := 0.7
	reason := "no parameters — simple invocation"
	if len(fn.Returns) == 0 {
		confidence = 0.4
		reason = "no parameters and no return value — side-effect only"
	}

	skipLine := ""
	if confidence < 0.6 {
		skipLine = "\tt.Skip(\"TODO: define expected behavior\")\n"
	}

	var callExpr string
	if fn.IsMethod && fn.Receiver != nil {
		// For methods, create a receiver instance
		recType := fn.Receiver.Type.Name
		if fn.Receiver.Type.IsPointer {
			recType = "*" + recType[1:] // remove leading *
		}
		callExpr = fmt.Sprintf("%s{}.%s()", recType, fn.Name)
	} else {
		callExpr = fmt.Sprintf("%s()", fn.Name)
	}

	code := fmt.Sprintf(`func Test%s_EdgeCases(t *testing.T) {
	_ = %s
%s}`, fn.Name, callExpr, skipLine)

	return &generator.Result{
		Code:       code,
		Confidence: confidence,
		Reason:     reason,
		Imports:    nil,
	}
}

// uniquify ensures paramName does not conflict with reserved names by appending _.
func (e *edgeGen) uniquify(paramName string, params []parser.ParamInfo) string {
	reserved := map[string]bool{"name": true, "desc": true}
	for _, p := range params {
		if p.Name != "" {
			reserved[p.Name] = true
		}
	}
	if !reserved[paramName] {
		return paramName
	}
	for i := 0; ; i++ {
		candidate := fmt.Sprintf("%s_%d", paramName, i)
		if !reserved[candidate] {
			return candidate
		}
	}
}

func (e *edgeGen) uniquifyAmong(name string, existing []string) string {
	seen := make(map[string]bool)
	for _, n := range existing {
		seen[n] = true
	}
	if !seen[name] {
		return name
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s%d", name, i)
		if !seen[candidate] {
			return candidate
		}
	}
}

func (e *edgeGen) score(fn *parser.FuncInfo) (float64, string) {
	if len(fn.Params) == 0 {
		if len(fn.Returns) == 0 {
			return 0.4, "no parameters and no returns — side-effect only"
		}
		return 0.7, "no parameters — simple invocation"
	}

	totalScore := 0.0
	knownCount := 0
	for _, p := range fn.Params {
		_, known := edgeMap[p.Type.Kind]
		if known || p.Type.Kind == parser.KindStruct || p.Type.Kind == parser.KindNamed {
			knownCount++
		}
	}

	// Type coverage factor (0–0.4)
	if len(fn.Params) > 0 {
		coverage := float64(knownCount) / float64(len(fn.Params))
		totalScore += coverage * 0.4
	}

	// Complexity factor (0–0.2): fewer params = simpler
	if len(fn.Params) <= 2 {
		totalScore += 0.2
	} else if len(fn.Params) <= 4 {
		totalScore += 0.1
	}

	// Return factor (0–0.2)
	if len(fn.Returns) > 0 {
		totalScore += 0.2
	}

	// Error factor (0–0.2): error in returns reduces confidence slightly
	hasError := false
	for _, r := range fn.Returns {
		if r.Name == "error" {
			hasError = true
			break
		}
	}
	if hasError {
		totalScore -= 0.05
	} else {
		totalScore += 0.2
	}

	// Method factor (0–0.1)
	if fn.IsMethod {
		totalScore += 0.1 // slightly more predictable
	}

	// Clamp
	if totalScore > 1.0 {
		totalScore = 1.0
	}
	if totalScore < 0.0 {
		totalScore = 0.0
	}

	reason := fmt.Sprintf("type coverage %.0f%%, %d params, %d returns", (totalScore/0.4)*100, len(fn.Params), len(fn.Returns))
	return totalScore, reason
}
