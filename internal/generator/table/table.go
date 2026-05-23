// Copyright (c) 2026 Kirill Scherba <kirill@scherba.ru>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package table generates table-driven test boilerplate for pure functions.
package table

import (
	"fmt"
	"strings"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
	"github.com/kirill-scherba/go-test-oracle/internal/parser"
	"github.com/kirill-scherba/go-test-oracle/internal/score"
)

// New creates a table-driven generator.
func New() generator.Generator { return &tableGen{} }

type tableGen struct{}

func (tg *tableGen) Name() string { return "table" }

// typicalValue returns a default input value for table-driven tests.
func typicalValue(t parser.TypeInfo) string {
	switch t.Kind {
	case parser.KindInt:
		return "42"
	case parser.KindFloat:
		return "3.14"
	case parser.KindString:
		return `"hello"`
	case parser.KindBool:
		return "true"
	case parser.KindSlice:
		return fmt.Sprintf("%s{1, 2, 3}", t.Name)
	case parser.KindMap:
		return fmt.Sprintf("%s{}", t.Name)
	case parser.KindPointer:
		return "nil"
	case parser.KindStruct:
		return fmt.Sprintf("%s{}", t.Name)
	case parser.KindNamed:
		return fmt.Sprintf("%s{}", t.Name)
	case parser.KindGeneric:
		return fmt.Sprintf("%s{}", t.Name)
	default:
		return fmt.Sprintf("%s{}", t.Name)
	}
}

// wantValue returns a valid Go zero-value expression for a return type
// in table-driven test expected values. For known interface types such as
// error, or for types that cannot be instantiated with a literal, it
// returns nil as a safe placeholder.
func wantValue(t parser.TypeInfo) string {
	switch t.Kind {
	case parser.KindInt:
		return "0"
	case parser.KindFloat:
		return "0.0"
	case parser.KindString:
		return `""`
	case parser.KindBool:
		return "false"
	case parser.KindSlice, parser.KindMap, parser.KindPointer,
		parser.KindChan, parser.KindFunc, parser.KindInterface:
		return "nil"
	case parser.KindStruct:
		return fmt.Sprintf("%s{}", t.Name)
	case parser.KindNamed:
		// Named types may be interfaces (e.g. error) — use nil as safe default.
		if t.Name == "error" {
			return "nil"
		}
		return fmt.Sprintf("%s{} /* TODO: ensure %s is a struct type */", t.Name, t.Name)
	case parser.KindGeneric:
		return fmt.Sprintf("%s{}", t.Name)
	default:
		return "nil"
	}
}

func (tg *tableGen) Generate(fn *parser.FuncInfo) (*generator.Result, error) {
	s := score.Calculate(fn)

	// If not a pure function, decline gracefully
	if !score.IsPureHeuristic(fn) {
		return &generator.Result{
			Code:       fmt.Sprintf("// %s is not a pure function — table-driven test skipped\n", fn.Name),
			Confidence: 0.0,
			Reason:     "not a pure function (has side effects or no returns)",
			Imports:    nil,
		}, nil
	}

	if len(fn.Params) == 0 {
		return tg.generateNoParams(fn, s), nil
	}

	// Build struct fields
	testNameField := "name"
	fields := []string{fmt.Sprintf("%s string", testNameField)}

	paramNames := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		n := p.Name
		if n == "" || n == "_" {
			n = fmt.Sprintf("arg%d", i)
		}
		// Uniquify against testNameField and previous params
		n = tg.uniquifyAmong(n, append([]string{testNameField}, paramNames[:i]...))
		paramNames[i] = n
		fields = append(fields, fmt.Sprintf("%s %s", n, p.Type.Name))
	}

	// Want field for each return value
	wantNames := make([]string, len(fn.Returns))
	for i, r := range fn.Returns {
		wn := fmt.Sprintf("want%d", i)
		if r.Name != "" {
			wn = "want_" + r.Name
		}
		wantNames[i] = wn
		fields = append(fields, fmt.Sprintf("%s %s", wn, r.Name))
	}

	// Build test cases — just two: empty/typical inputs
	cases := []string{}
	for i, name := range []string{"empty", "typical"} {
		vals := make([]string, len(fn.Params))
		for j, p := range fn.Params {
			if i == 0 {
				// "empty" case: zero values
				vals[j] = zeroValue(p.Type)
			} else {
				vals[j] = typicalValue(p.Type)
			}
		}

		wants := make([]string, len(fn.Returns))
		for j := range fn.Returns {
			wants[j] = wantValue(fn.Returns[j])
		}

		lines := []string{
			fmt.Sprintf("\t\t\t%s: %q,", testNameField, name),
		}
		for j, v := range vals {
			lines = append(lines, fmt.Sprintf("\t\t\t%s: %s,", paramNames[j], v))
		}
		for j, w := range wants {
			lines = append(lines, fmt.Sprintf("\t\t\t%s: %s,", wantNames[j], w))
		}
		cases = append(cases, fmt.Sprintf("\t\t{\n%s\n\t\t},", strings.Join(lines, "\n")))
	}

	// Build function call
	callArgs := make([]string, len(paramNames))
	for i, n := range paramNames {
		callArgs[i] = fmt.Sprintf("tt.%s", n)
	}
	callExpr := fmt.Sprintf("%s(%s)", fn.Name, strings.Join(callArgs, ", "))

	// Handle methods
	if fn.IsMethod && fn.Receiver != nil {
		recType := fn.Receiver.Type.Name
		if fn.Receiver.Type.IsPointer {
			recType = strings.TrimPrefix(recType, "*")
		}
		callExpr = fmt.Sprintf("%s{}.%s(%s)", recType, fn.Name, strings.Join(callArgs, ", "))
	}

	// Build return comparison
	gotVars := make([]string, len(fn.Returns))
	for i := range fn.Returns {
		gotVars[i] = fmt.Sprintf("got%d", i)
	}
	assignLine := fmt.Sprintf("%s := %s", strings.Join(gotVars, ", "), callExpr)

	checkLines := []string{}
	for i, wn := range wantNames {
		checkLines = append(checkLines, fmt.Sprintf("\t\tif got%d != tt.%s {", i, wn))
		checkLines = append(checkLines, fmt.Sprintf("\t\t\tt.Errorf(\"%s() got%%v want%%v\", got%d, tt.%s)", fn.Name, i, wn))
		checkLines = append(checkLines, "\t\t}")
	}

	code := fmt.Sprintf(`func Test%s(t *testing.T) {
	tests := []struct {
%s
	}{
%s
	}
	for _, tt := range tests {
		t.Run(tt.%s, func(t *testing.T) {
			%s
%s
		})
	}
}`, fn.Name, strings.Join(fields, "\n"), strings.Join(cases, "\n"), testNameField, assignLine, strings.Join(checkLines, "\n"))

	return &generator.Result{
		Code:       code,
		Confidence: s.Confidence,
		Reason:     s.Reason + " (table-driven)",
		Imports:    nil,
	}, nil
}

func (tg *tableGen) generateNoParams(fn *parser.FuncInfo, s score.Score) *generator.Result {
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

	gotVars := make([]string, len(fn.Returns))
	for i := range fn.Returns {
		gotVars[i] = fmt.Sprintf("got%d", i)
	}
	assignLine := fmt.Sprintf("%s := %s", strings.Join(gotVars, ", "), callExpr)

	checkLines := []string{}
	for i := range fn.Returns {
		checkLines = append(checkLines, fmt.Sprintf("\t\tif got%d == nil {", i))
		checkLines = append(checkLines, "\t\t\tt.Skip(\"TODO: define expected behavior\")")
		checkLines = append(checkLines, "\t\t}")
	}

	code := fmt.Sprintf(`func Test%s(t *testing.T) {
	%s
%s
}`, fn.Name, assignLine, strings.Join(checkLines, "\n"))

	return &generator.Result{
		Code:       code,
		Confidence: s.Confidence,
		Reason:     s.Reason + " (table-driven no-params)",
		Imports:    nil,
	}
}

func zeroValue(t parser.TypeInfo) string {
	switch t.Kind {
	case parser.KindInt:
		return "0"
	case parser.KindFloat:
		return "0.0"
	case parser.KindString:
		return `""`
	case parser.KindBool:
		return "false"
	case parser.KindSlice, parser.KindMap, parser.KindPointer, parser.KindChan, parser.KindFunc:
		return "nil"
	case parser.KindStruct, parser.KindNamed, parser.KindGeneric:
		return fmt.Sprintf("%s{}", t.Name)
	default:
		return fmt.Sprintf("%s{}", t.Name)
	}
}

func (tg *tableGen) uniquifyAmong(name string, existing []string) string {
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
