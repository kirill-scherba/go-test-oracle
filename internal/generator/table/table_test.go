package table

import (
	"strings"
	"testing"

	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

func TestTableGenerator_SimpleFunc(t *testing.T) {
	gen := New()
	fn := &parser.FuncInfo{
		Name: "Add",
		Params: []parser.ParamInfo{
			{Name: "a", Type: parser.TypeInfo{Name: "int", Kind: parser.KindInt}},
			{Name: "b", Type: parser.TypeInfo{Name: "int", Kind: parser.KindInt}},
		},
		Returns:    []parser.TypeInfo{{Name: "int", Kind: parser.KindInt}},
		IsExported: true,
	}

	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(res.Code, "TestAdd") {
		t.Error("generated code missing TestAdd")
	}
	if !strings.Contains(res.Code, "for _, tt := range tests") {
		t.Error("generated code missing table loop")
	}
	if !strings.Contains(res.Code, "t.Run(tt.name") {
		t.Error("generated code missing subtest naming")
	}
	if !strings.Contains(res.Code, "Add(") {
		t.Error("generated code missing Add call")
	}
	if !strings.Contains(res.Code, `t.Skip("TODO: define expected behavior")`) {
		t.Error("generated table test should skip placeholder expected values")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestTableGenerator_SkipsImpure(t *testing.T) {
	gen := New()
	fn := &parser.FuncInfo{
		Name: "Mutate",
		Params: []parser.ParamInfo{
			{Name: "p", Type: parser.TypeInfo{Kind: parser.KindPointer, IsPointer: true, Elem: &parser.TypeInfo{Kind: parser.KindStruct}}},
		},
		Returns:    []parser.TypeInfo{{Kind: parser.KindBool}},
		IsExported: true,
	}

	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(res.Code, "not a pure function") {
		t.Error("generated code should mention non-pure function")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestTableGenerator_NoParams(t *testing.T) {
	gen := New()
	fn := &parser.FuncInfo{
		Name:       "NoParams",
		IsExported: true,
		Returns:    []parser.TypeInfo{{Name: "int", Kind: parser.KindInt}},
	}

	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(res.Code, "TestNoParams") {
		t.Error("generated code missing TestNoParams")
	}
	if !strings.Contains(res.Code, "NoParams()") {
		t.Error("generated code missing NoParams() call")
	}
	if strings.Contains(res.Code, "== nil") {
		t.Error("generated no-params table test should not compare arbitrary return values with nil")
	}
	if !strings.Contains(res.Code, `t.Skip("TODO: define expected behavior")`) {
		t.Error("generated no-params table test should skip placeholder expected behavior")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestTableGenerator_Method(t *testing.T) {
	gen := New()
	fn := &parser.FuncInfo{
		Name:       "Method",
		IsExported: true,
		IsMethod:   true,
		Receiver: &parser.ParamInfo{
			Name: "c",
			Type: parser.TypeInfo{Name: "Counter", Kind: parser.KindStruct},
		},
		Params: []parser.ParamInfo{
			{Name: "n", Type: parser.TypeInfo{Name: "int", Kind: parser.KindInt}},
		},
		Returns: []parser.TypeInfo{{Name: "int", Kind: parser.KindInt}},
	}

	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(res.Code, "TestMethod") {
		t.Error("generated code missing TestMethod")
	}
	if !strings.Contains(res.Code, "Counter{}.") {
		t.Error("generated code missing Counter{} receiver")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestTableGenerator_ErrorReturn(t *testing.T) {
	gen := New()
	fn := &parser.FuncInfo{
		Name: "MayFail",
		Params: []parser.ParamInfo{
			{Name: "input", Type: parser.TypeInfo{Name: "string", Kind: parser.KindString}},
		},
		Returns: []parser.TypeInfo{
			{Name: "string", Kind: parser.KindString},
			{Name: "error", Kind: parser.KindNamed},
		},
		IsExported: true,
	}

	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(res.Code, "TestMayFail") {
		t.Error("generated code missing TestMayFail")
	}
	if !strings.Contains(res.Code, "want_error") {
		t.Error("generated code missing want_error field")
	}

	t.Logf("generated:\n%s", res.Code)
}
