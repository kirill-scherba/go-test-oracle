package fuzz

import (
	"strings"
	"testing"

	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

func TestFuzzGenerator_SimpleFunc(t *testing.T) {
	gen := New()
	fn := &parser.FuncInfo{
		Name: "SimpleFunc",
		Params: []parser.ParamInfo{
			{Name: "name", Type: parser.TypeInfo{Name: "string", Kind: parser.KindString}},
			{Name: "count", Type: parser.TypeInfo{Name: "int", Kind: parser.KindInt}},
		},
		Returns:    []parser.TypeInfo{{Name: "string", Kind: parser.KindString}},
		IsExported: true,
	}

	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(res.Code, "FuzzSimpleFunc") {
		t.Error("generated code missing FuzzSimpleFunc")
	}
	if !strings.Contains(res.Code, "f.Fuzz(func(t *testing.T") {
		t.Error("generated code missing f.Fuzz signature")
	}
	if !strings.Contains(res.Code, "f.Add(") {
		t.Error("generated code missing seed corpus")
	}
	if !strings.Contains(res.Code, "SimpleFunc(") {
		t.Error("generated code missing SimpleFunc call")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestFuzzGenerator_FloatFunc(t *testing.T) {
	gen := New()
	fn := &parser.FuncInfo{
		Name: "FloatFunc",
		Params: []parser.ParamInfo{
			{Name: "v", Type: parser.TypeInfo{Name: "float64", Kind: parser.KindFloat}},
		},
		Returns:    []parser.TypeInfo{{Name: "float64", Kind: parser.KindFloat}},
		IsExported: true,
	}

	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(res.Code, "FuzzFloatFunc") {
		t.Error("generated code missing FuzzFloatFunc")
	}
	if !strings.Contains(res.Code, "float64") {
		t.Error("generated code missing float64 type")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestFuzzGenerator_NoParams(t *testing.T) {
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

	if !strings.Contains(res.Code, "FuzzNoParams") {
		t.Error("generated code missing FuzzNoParams")
	}
	if !strings.Contains(res.Code, "NoParams()") {
		t.Error("generated code missing NoParams() call")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestFuzzGenerator_Method(t *testing.T) {
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

	if !strings.Contains(res.Code, "FuzzMethod") {
		t.Error("generated code missing FuzzMethod")
	}
	if !strings.Contains(res.Code, "Counter{}.") {
		t.Error("generated code missing Counter{} receiver")
	}

	t.Logf("generated:\n%s", res.Code)
}
