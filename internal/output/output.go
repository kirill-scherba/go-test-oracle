// Copyright (c) 2026 Kirill Scherba <kirill@scherba.ru>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package output formats generated test code into different output modes.
package output

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
)

// knownTestImports lists imports that should always be included in file output.
var knownTestImports = []string{
	"testing",
}

// Format turns a generator result into the requested output format.
func Format(result *generator.Result, mode string, pkg string, existingImports []string) string {
	switch mode {
	case "file":
		return formatFile(result, pkg, existingImports)
	case "diff":
		return formatDiff(result, pkg)
	default: // fragment
		return result.Code
	}
}

// formatFile wraps the generated code in a complete *_test.go file.
func formatFile(result *generator.Result, pkg string, existingImports []string) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("package %s", pkg))
	parts = append(parts, "")

	// Imports
	allImports := append(existingImports, result.Imports...)
	allImports = mergeImports(allImports)
	if len(allImports) > 0 {
		parts = append(parts, "import (")
		for _, imp := range allImports {
			parts = append(parts, "\t\""+imp+"\"")
		}
		parts = append(parts, ")")
		parts = append(parts, "")
	}

	parts = append(parts, result.Code)
	return strings.Join(parts, "\n")
}

// formatDiff generates a unified diff snippet.
func formatDiff(result *generator.Result, pkg string) string {
	// For simplicity, output as a "new file" diff
	code := formatFile(result, pkg, nil)
	return toDiff(code, pkg)
}

// FormatFileAll wraps multiple code fragments into a single *_test.go file
// with one package declaration and a merged import block.
func FormatFileAll(fragments []string, pkg string, extraImports []string) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("package %s", pkg))
	parts = append(parts, "")

	allImports := append([]string{}, knownTestImports...)
	allImports = append(allImports, extraImports...)
	allImports = mergeImports(allImports)
	if len(allImports) > 0 {
		parts = append(parts, "import (")
		for _, imp := range allImports {
			parts = append(parts, "\t\""+imp+"\"")
		}
		parts = append(parts, ")")
		parts = append(parts, "")
	}

	for i, f := range fragments {
		if i > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, f)
	}
	return strings.Join(parts, "\n")
}

// FormatDiffAll wraps multiple code fragments into a unified diff.
func FormatDiffAll(fragments []string, pkg string, extraImports []string) string {
	code := FormatFileAll(fragments, pkg, extraImports)
	return toDiff(code, pkg)
}

// toDiff converts source code to a unified diff format.
func toDiff(code string, pkg string) string {
	lines := strings.Split(code, "\n")
	var out []string
	out = append(out, fmt.Sprintf("--- /dev/null\n+++ %s_test.go", pkg))
	out = append(out, fmt.Sprintf("@@ -0,0 +1,%d @@", len(lines)))
	for _, line := range lines {
		out = append(out, "+"+line)
	}
	return strings.Join(out, "\n")
}

// mergeImports deduplicates and sorts import paths.
// It always ensures "testing" is present when using file output.
func mergeImports(imports []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, imp := range imports {
		if imp != "" && !seen[imp] {
			seen[imp] = true
			result = append(result, imp)
		}
	}
	sort.Strings(result)
	return result
}

// TestFileName returns the appropriate test file path for a source file.
func TestFileName(srcPath string) string {
	ext := filepath.Ext(srcPath)
	base := strings.TrimSuffix(srcPath, ext)
	return base + "_test" + ext
}
