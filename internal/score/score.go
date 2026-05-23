// Copyright (c) 2026 Kirill Scherba <kirill@scherba.ru>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package score calculates confidence scores for generated tests.
package score

import (
	"fmt"

	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

// Score holds the confidence evaluation result.
type Score struct {
	Confidence float64            // 0.0–1.0
	Factors    map[string]float64 // Per-factor contributions
	Reason     string             // Human-readable explanation
	Skip       bool               // Inject t.Skip if true
}

// Threshold values for confidence scoring.
const (
	ThresholdHigh   = 0.8 // Confidence above this is considered high
	ThresholdMedium = 0.6 // Confidence above this is considered medium
	ThresholdLow    = 0.0 // Confidence below this is considered low
)

// knownEdgeTypes counts how many parameter types have defined edge cases.
var knownEdgeTypes = map[parser.TypeKind]bool{
	parser.KindBool:      true,
	parser.KindInt:       true,
	parser.KindFloat:     true,
	parser.KindString:    true,
	parser.KindSlice:     true,
	parser.KindMap:       true,
	parser.KindPointer:   true,
	parser.KindChan:      true,
	parser.KindInterface: true,
	parser.KindFunc:      true,
}

// Calculate computes confidence for a function based on heuristics.
func Calculate(fn *parser.FuncInfo) Score {
	s := Score{
		Factors: make(map[string]float64),
	}

	if len(fn.Params) == 0 {
		if len(fn.Returns) == 0 {
			s.Confidence = 0.4
			s.Reason = "no parameters and no returns — side-effect only"
			s.Skip = true
			return s
		}
		s.Confidence = 0.7
		s.Reason = "no parameters — simple invocation"
		s.Skip = false
		return s
	}

	total := 0.0

	// 1. Type coverage factor (weight 0.40)
	knownCount := 0
	for _, p := range fn.Params {
		if knownEdgeTypes[p.Type.Kind] || p.Type.Kind == parser.KindStruct || p.Type.Kind == parser.KindNamed {
			knownCount++
		}
	}
	coverage := float64(knownCount) / float64(len(fn.Params))
	total += coverage * 0.40
	s.Factors["type_coverage"] = coverage * 0.40

	// 2. Complexity factor (weight 0.20): fewer params = simpler
	if len(fn.Params) <= 2 {
		total += 0.20
		s.Factors["complexity"] = 0.20
	} else if len(fn.Params) <= 4 {
		total += 0.10
		s.Factors["complexity"] = 0.10
	} else {
		s.Factors["complexity"] = 0.0
	}

	// 3. Return factor (weight 0.20)
	if len(fn.Returns) > 0 {
		total += 0.20
		s.Factors["returns"] = 0.20
	} else {
		s.Factors["returns"] = 0.0
	}

	// 4. Error handling factor (weight 0.20): error in returns complicates things
	hasError := false
	for _, r := range fn.Returns {
		if r.Name == "error" {
			hasError = true
			break
		}
	}
	if hasError {
		total -= 0.05
		s.Factors["error_handling"] = -0.05
	} else {
		total += 0.20
		s.Factors["error_handling"] = 0.20
	}

	// 5. Method factor (weight 0.10): methods slightly more predictable
	if fn.IsMethod {
		total += 0.10
		s.Factors["method"] = 0.10
	}

	// Clamp
	if total > 1.0 {
		total = 1.0
	}
	if total < 0.0 {
		total = 0.0
	}

	s.Confidence = total
	s.Skip = total < ThresholdMedium
	s.Reason = fmt.Sprintf("type coverage %.0f%%, %d params, %d returns, confidence %.2f",
		coverage*100, len(fn.Params), len(fn.Returns), total)

	return s
}

// IsPureHeuristic returns true if the function is likely a pure function.
// Used by table-driven generator to decide whether table-driven is appropriate.
func IsPureHeuristic(fn *parser.FuncInfo) bool {
	// Side effects: pointer params (mutation), channels, context, etc.
	for _, p := range fn.Params {
		// Pointer to mutable struct — potential side effect
		if p.Type.IsPointer && p.Type.Elem != nil && p.Type.Elem.Kind == parser.KindStruct {
			return false
		}
		// Channel, mutex, context, etc. — side effect
		if p.Type.IsChan || p.Type.Kind == parser.KindInterface {
			return false
		}
	}

	// No return values — side effect only
	if len(fn.Returns) == 0 {
		return false
	}

	return true
}
