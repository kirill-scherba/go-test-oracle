// Copyright (c) 2026 Kirill Scherba <kirill@scherba.ru>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// go-test-oracle CLI — generates Go test scaffolding from source code.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/edge"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/fuzz"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/table"
	"github.com/kirill-scherba/go-test-oracle/internal/output"
	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

// generatorsFlag implements the flag.Value interface for comma-separated
// generator names.
type generatorsFlag []string

func (g *generatorsFlag) String() string { return strings.Join(*g, ",") }
func (g *generatorsFlag) Set(v string) error {
	for _, s := range strings.Split(v, ",") {
		*g = append(*g, strings.TrimSpace(s))
	}
	return nil
}

func main() {
	var (
		funcFlag      = flag.String("func", "", "Function name to generate tests for (default: all exported)")
		generatorFlag = generatorsFlag{} // empty default means all generators
		outputFlag    = flag.String("output", "stdout", "Output target: stdout (default), file, or path to write")
		formatFlag    = flag.String("format", "fragment", "Output format: fragment (default), file, diff")
		allFlag       = flag.Bool("all", false, "Include unexported functions")
	)
	flag.Var(&generatorFlag, "generator", "Generator(s): edge, fuzz, table, all (default)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: go-test-oracle [flags] <file.go> [function]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	path := args[0]
	if *funcFlag == "" && len(args) > 1 {
		*funcFlag = args[1]
	}

	// Parse source file and extract package name
	funcs, err := parser.ParseFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", path, err)
		os.Exit(1)
	}
	pkgName, err := parser.PackageName(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading package name from %s: %v\n", path, err)
		os.Exit(1)
	}

	// Build generator list
	gens := selectGenerators(generatorFlag)

	// Filter functions
	var targets []parser.FuncInfo
	if *funcFlag != "" {
		for _, fn := range funcs {
			if fn.Name == *funcFlag {
				targets = append(targets, fn)
				break
			}
		}
		if len(targets) == 0 {
			fmt.Fprintf(os.Stderr, "Function %q not found in %s\n", *funcFlag, path)
			os.Exit(1)
		}
	} else {
		for _, fn := range funcs {
			if fn.IsExported || *allFlag {
				targets = append(targets, fn)
			}
		}
	}

	// Collect generation results
	type genResult struct {
		gen    string
		fn     string
		result *generator.Result
	}
	var results []genResult
	for _, fn := range targets {
		for _, gen := range gens {
			res, err := gen.Generate(&fn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating %s for %s: %v\n", gen.Name(), fn.Name, err)
				continue
			}
			if res == nil {
				continue
			}
			results = append(results, genResult{gen.Name(), fn.Name, res})
		}
	}

	// Build output
	var fragments []string
	var allImports []string
	for _, r := range results {
		fragments = append(fragments, fmt.Sprintf(
			"// --- %s: %s (confidence: %.2f)\n// %s\n%s",
			r.gen, r.fn, r.result.Confidence, r.result.Reason, r.result.Code,
		))
		allImports = append(allImports, r.result.Imports...)
	}

	outputBody := strings.Join(fragments, "\n\n")

	// Apply format wrapping
	switch *formatFlag {
	case "file":
		outputBody = output.FormatFileAll(fragments, pkgName, allImports)
	case "diff":
		outputBody = output.FormatDiffAll(fragments, pkgName, allImports)
	}

	switch *outputFlag {
	case "stdout":
		fmt.Println(outputBody)
	case "file":
		testPath := output.TestFileName(path)
		if err := os.WriteFile(testPath, []byte(outputBody), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", testPath, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote %s\n", testPath)
	default:
		// Treat as file path
		if err := os.WriteFile(*outputFlag, []byte(outputBody), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *outputFlag, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote %s\n", *outputFlag)
	}
}

// selectGenerators returns the generator instances matching the requested names.
// An empty or ["all"] list returns all available generators.
func selectGenerators(names []string) []generator.Generator {
	all := []generator.Generator{edge.New(), fuzz.New(), table.New()}
	if len(names) == 0 {
		return all
	}
	for _, name := range names {
		if name == "all" {
			return all
		}
	}

	var selected []generator.Generator
	for _, name := range names {
		switch name {
		case "edge":
			selected = append(selected, edge.New())
		case "fuzz":
			selected = append(selected, fuzz.New())
		case "table":
			selected = append(selected, table.New())
		}
	}
	if len(selected) == 0 {
		return all
	}
	return selected
}
