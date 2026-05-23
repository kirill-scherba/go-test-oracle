// Package output formats generated test code into different output modes.
package output

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
)

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
	lines := strings.Split(code, "\n")
	var out []string
	out = append(out, fmt.Sprintf("--- /dev/null\n+++ %s_test.go", pkg))
	out = append(out, fmt.Sprintf("@@ -0,0 +1,%d @@", len(lines)))
	for _, line := range lines {
		out = append(out, "+"+line)
	}
	return strings.Join(out, "\n")
}

// TestFileName returns the appropriate test file path for a source file.
func TestFileName(srcPath string) string {
	ext := filepath.Ext(srcPath)
	base := strings.TrimSuffix(srcPath, ext)
	return base + "_test" + ext
}
