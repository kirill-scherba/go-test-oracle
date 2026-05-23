// go-test-oracle CLI — generates Go test scaffolding from source code.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/edge"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/fuzz"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/table"
	"github.com/kirill-scherba/go-test-oracle/internal/output"
	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

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
		generatorFlag = generatorsFlag{"all"}
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

	// Determine package name from file path
	pkg := filepath.Base(filepath.Dir(path))
	if pkg == "." || pkg == "" {
		pkg = "main"
	}

	// Parse source file
	funcs, err := parser.ParseFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", path, err)
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

	var out []string
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

			// Collect header
			out = append(out, fmt.Sprintf("// --- %s: %s (confidence: %.2f)\n// %s\n", gen.Name(), fn.Name, res.Confidence, res.Reason))
			out = append(out, output.Format(res, *formatFlag, pkg, nil))
			out = append(out, "")
		}
	}

	result := strings.Join(out, "\n")

	switch *outputFlag {
	case "stdout":
		fmt.Println(result)
	case "file":
		testPath := output.TestFileName(path)
		if err := os.WriteFile(testPath, []byte(result), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", testPath, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote %s\n", testPath)
	default:
		// Treat as file path
		if err := os.WriteFile(*outputFlag, []byte(result), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *outputFlag, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote %s\n", *outputFlag)
	}
}

func selectGenerators(names []string) []generator.Generator {
	all := []generator.Generator{edge.New(), fuzz.New(), table.New()}
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
