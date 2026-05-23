package edge

import (
	"strings"
	"testing"

	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

func TestEdgeGenerator_SimpleFunc(t *testing.T) {
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

	// Must contain function name
	if !strings.Contains(res.Code, "TestSimpleFunc_EdgeCases") {
		t.Error("generated code missing TestSimpleFunc_EdgeCases")
	}

	// Must contain edge case names for both params
	for _, want := range []string{"name_empty", "name_space", "name_unicode", "count_zero", "count_negative", "count_max"} {
		if !strings.Contains(res.Code, want) {
			t.Errorf("generated code missing edge case %q", want)
		}
	}

	// Must call the function under test
	if !strings.Contains(res.Code, "SimpleFunc(") {
		t.Error("generated code missing SimpleFunc call")
	}

	// Should have reasonable confidence
	if res.Confidence < 0.5 || res.Confidence > 1.0 {
		t.Errorf("confidence = %.2f, want in [0.5,1.0]", res.Confidence)
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestEdgeGenerator_NoParams(t *testing.T) {
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

	if !strings.Contains(res.Code, "TestNoParams_EdgeCases") {
		t.Error("generated code missing TestNoParams_EdgeCases")
	}
	if !strings.Contains(res.Code, "NoParams()") {
		t.Error("generated code missing NoParams() call")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestEdgeGenerator_PointerParam(t *testing.T) {
	gen := New()
	fn := &parser.FuncInfo{
		Name: "PointerParam",
		Params: []parser.ParamInfo{
			{Name: "data", Type: parser.TypeInfo{Name: "*string", Kind: parser.KindPointer, IsPointer: true}},
		},
		Returns:    []parser.TypeInfo{{Name: "bool", Kind: parser.KindBool}},
		IsExported: true,
	}

	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(res.Code, "data_nil") {
		t.Error("generated code missing data_nil edge case")
	}
	if !strings.Contains(res.Code, "tt.data") {
		t.Error("generated code missing tt.data reference")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestEdgeGenerator_LowConfidence(t *testing.T) {
	gen := New()
	// Function with no params and no returns — lowest confidence
	fn := &parser.FuncInfo{
		Name:       "SideEffectOnly",
		IsExported: true,
		Params:     nil,
		Returns:    nil,
	}

	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if res.Confidence >= 0.6 {
		t.Errorf("confidence = %.2f, want < 0.6 for no-params no-returns", res.Confidence)
	}
	if !strings.Contains(res.Code, `t.Skip("TODO: define expected behavior")`) {
		t.Error("low-confidence test should contain t.Skip")
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestEdgeGenerator_FloatParam(t *testing.T) {
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

	for _, want := range []string{"math.NaN()", "math.Inf(1)", "math.Inf(-1)", "math.MaxInt64"} {
		if strings.Contains(res.Code, want) {
			// At least one of these math constants should be present
			break
		}
		if want == "math.MaxInt64" {
			t.Error("generated code missing math constants for float edge cases")
		}
	}

	t.Logf("generated:\n%s", res.Code)
}

func TestEdgeGenerator_Method(t *testing.T) {
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

	if !strings.Contains(res.Code, "TestMethod_EdgeCases") {
		t.Error("generated code missing TestMethod_EdgeCases")
	}

	t.Logf("generated:\n%s", res.Code)
}
