// Copyright (c) 2026 Kirill Scherba <kirill@scherba.ru>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package generator defines the test generator interface.
package generator

import (
	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

// Result is the output of a test generator.
type Result struct {
	Code       string   // Generated Go test code
	Confidence float64  // 0.0–1.0
	Reason     string   // Human-readable explanation
	Imports    []string // Additional imports needed
}

// Generator creates test code from a FuncInfo.
type Generator interface {
	Name() string
	Generate(fn *parser.FuncInfo) (*Result, error)
}
