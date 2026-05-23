package score

import (
	"testing"

	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

func TestCalculate(t *testing.T) {
	cases := []struct {
		name     string
		fn       *parser.FuncInfo
		wantMin  float64
		wantMax  float64
		wantSkip bool
	}{
		{
			name: "simple pure function",
			fn: &parser.FuncInfo{
				Name: "Add",
				Params: []parser.ParamInfo{
					{Name: "a", Type: parser.TypeInfo{Kind: parser.KindInt}},
					{Name: "b", Type: parser.TypeInfo{Kind: parser.KindInt}},
				},
				Returns: []parser.TypeInfo{{Kind: parser.KindInt}},
			},
			wantMin:  0.8,
			wantMax:  1.0,
			wantSkip: false,
		},
		{
			name: "side-effect only (no params no returns)",
			fn: &parser.FuncInfo{
				Name: "DoSomething",
			},
			wantMin:  0.0,
			wantMax:  0.5,
			wantSkip: true,
		},
		{
			name: "pointer param (uncertain)",
			fn: &parser.FuncInfo{
				Name: "Mutate",
				Params: []parser.ParamInfo{
					{Name: "p", Type: parser.TypeInfo{Kind: parser.KindPointer, IsPointer: true}},
				},
				Returns: []parser.TypeInfo{{Kind: parser.KindBool}},
			},
			wantMin:  0.9,
			wantMax:  1.0,
			wantSkip: false,
		},
		{
			name: "many params (complex)",
			fn: &parser.FuncInfo{
				Name: "Complex",
				Params: []parser.ParamInfo{
					{Name: "a", Type: parser.TypeInfo{Kind: parser.KindInt}},
					{Name: "b", Type: parser.TypeInfo{Kind: parser.KindInt}},
					{Name: "c", Type: parser.TypeInfo{Kind: parser.KindInt}},
					{Name: "d", Type: parser.TypeInfo{Kind: parser.KindInt}},
					{Name: "e", Type: parser.TypeInfo{Kind: parser.KindInt}},
				},
				Returns: []parser.TypeInfo{{Kind: parser.KindInt}},
			},
			wantMin:  0.7,
			wantMax:  0.9,
			wantSkip: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Calculate(tc.fn)
			if s.Confidence < tc.wantMin || s.Confidence > tc.wantMax {
				t.Errorf("confidence = %.2f, want in [%.2f, %.2f]", s.Confidence, tc.wantMin, tc.wantMax)
			}
			if s.Skip != tc.wantSkip {
				t.Errorf("Skip = %v, want %v", s.Skip, tc.wantSkip)
			}
			t.Logf("reason: %s", s.Reason)
		})
	}
}

func TestIsPureHeuristic(t *testing.T) {
	pure := &parser.FuncInfo{
		Name: "Add",
		Params: []parser.ParamInfo{
			{Name: "a", Type: parser.TypeInfo{Kind: parser.KindInt}},
			{Name: "b", Type: parser.TypeInfo{Kind: parser.KindInt}},
		},
		Returns: []parser.TypeInfo{{Kind: parser.KindInt}},
	}
	if !IsPureHeuristic(pure) {
		t.Error("Add should be pure")
	}

	impure := &parser.FuncInfo{
		Name: "Mutate",
		Params: []parser.ParamInfo{
			{Name: "p", Type: parser.TypeInfo{Kind: parser.KindPointer, IsPointer: true, Elem: &parser.TypeInfo{Kind: parser.KindStruct}}},
		},
	}
	if IsPureHeuristic(impure) {
		t.Error("Mutate with pointer should not be pure")
	}
}
